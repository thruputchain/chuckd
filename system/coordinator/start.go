package coordinator

import (
	"context"
	"time"

	"github.com/thruputchain/chuckd/sub/sched"
	"github.com/thruputchain/chuckd/sub/scp/scpd"
	"github.com/thruputchain/chuckd/util/protoutil"
	"github.com/thruputchain/chuckd/util/xdrutil"
	"github.com/thruputchain/lib-go/liblog"
	"github.com/thruputchain/lib-go/libxdr"
	"github.com/thruputchain/lib-go/must"
	proto "github.com/thruputchain/thruput-protocol-go"
)

// System dependencies:
//	[] means delayed set after init
//
//	 coord
//
//	 discovery
//	 router --> discovery
//	 dogwood --> discovery, router
//
//	 catchup --> db
//
//	 multiplex --> dogwood
//	 submitter --> multiplex
//
//	 live --> [ scp_control ], coord (db)
//	 baton --> live
//	 [ scp_handler --> baton ]
//
//	 [ multiplex_receiver --> submitter, scp_control ]
//
//	 wallet --> submitter
//
// Delayed connections:
//
//	 live --> scp_control
//	 multiplex_receiver --> scp_control
//	 scp_handler --> multiplex, discovery, baton
//
//	 multiplex_receiver --> submitter, scp_control

type OperationalConfig struct {
	StorageDir     string
	SchedAddr      string
	DebugPort      int // used by coordinator bench only
	InAddr         string
	OutAddr        string
	SCPDaemonPath  string
	SCPHandleAddr  string
	SCPControlAddr string
	IsValidator    bool
	NetworkHash    *proto.NetworkHash
}

func (s *System) Start(ctx context.Context, opcfg OperationalConfig) Control {

	s.Storage.Start(s.ctx, opcfg.StorageDir)

	// start internal rpcs before connecting to scheduler, so scheduler can connect back
	s.insideSrv.Start(s.ctx, s.inside, opcfg.InAddr) // serves: coord_sched.x, wallet.x

	schedConn, err := sched.ConnectToScheduler(s.ctx, s.inside, opcfg.SchedAddr, opcfg.InAddr)
	must.NoError(ctx, err)
	s.SchedConn = schedConn

	s.RepairService.SetSchedConn(schedConn)
	s.Live.SetScheduler(schedConn)

	s.UnvettedSubmitter.Start(s.ctx, schedConn)

	// handshake with scpd
	scpHandler, scpController, scpdKill := scpd.StartSCPDaemon(
		s.ctx,
		opcfg.SCPDaemonPath,
		opcfg.SCPControlAddr,
		opcfg.SCPHandleAddr,
		opcfg.IsValidator,
		*s.cfg.Scp_config,
		protoutil.ToSecretKey(s.cfg.Validator_config.Private_info.Secret_key),
		opcfg.NetworkHash,
	)
	s.scpHandler = scpHandler
	s.scpController = scpController
	s.scpdKill = scpdKill

	// broker must be ready for operation before the baton starts
	s.SCPBroker.SetSCPController(scpController)

	s.Discovery.Start(s.ctx, scpController)
	scpHandler.SetHandlers(s.Multiplex, s.Discovery, s.Baton.SCPHandler())

	s.Dogwood.Start(s.ctx) // dogwood

	s.outsideSrv.Start(s.ctx, s.outside, opcfg.OutAddr) // discovery, dogwood, catchup, repair

	s.Dogwood.WaitReady(s.ctx) // wait until routes have been disseminated

	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		select {
		case <-s.ctx.Done():
			return
		case <-s.genesisReady: // wait for genesis to be in the db
		}
		liblog.Infof(ctx, "genesis ready")

		s.Baton.Start(s.ctx)

		// drive vetted transactions to dissemination
		s.wg.Add(1)
		go func() {
			defer s.wg.Done()
			buf := NewBatchingBufferForParallelSignedTxs(
				s.ctx,
				s.time,
				int(s.cfg.Mempool_config.Submission_lot_size),
				time.Duration(s.cfg.Mempool_config.Submission_lot_wait_msec)*time.Millisecond,
				func(ctx context.Context, batch proto.ParallelSignedTransactionBatch) {
					liblog.Debugf(ctx, "disseminating %d vetted txs", xdrutil.TokenSizeOfBatch(batch))
					s.Multiplex.SendSubmittedTxs(
						ctx,
						&proto.SubmittedTransactions{Transactions: batch},
					)
				},
			)
			for {
				batch, ok := s.CoordScheduler.ReceiveVetted(s.ctx)
				if !ok {
					return
				}
				liblog.Debugf(ctx, "received %d vetted txs from scheduler", xdrutil.TokenSizeOfBatch(batch))
				buf.Submit(s.ctx, batch)
			}
		}()

		// drive received envelopes into scp_control
		s.RepairService.SetSCPBroker(s.SCPBroker)
		s.wg.Add(1)
		go func() {
			defer s.wg.Done()
			for {
				_, env, ok := s.Multiplex.ReceiveEnvelope(s.ctx)
				if !ok {
					return
				}
				s.SCPBroker.SCPFeedEnvelope(s.ctx, *env)
			}
		}()

		// drive received txs into mempool
		s.wg.Add(1)
		go func() {
			defer s.wg.Done()
			m := libxdr.NewXdrMarshaler()
			_ = m
			for {
				sender, submittedTxs, ok := s.Multiplex.ReceiveSubmittedTxs(s.ctx)
				if !ok {
					return
				}
				_ = sender

				// liblog.Debugf("received %d txs from %v: %v",
				// 	len(submittedTxs.Transactions),
				// 	protoutil.VIDString(sender),
				// 	form.Display(protoutil.HashSignedTxs(m, submittedTxs.Transactions)))

				err := schedConn.Schedulercoord_submit_vetted(
					s.ctx,
					proto.ValidatorID{}, // XXX: who vetted them is not known
					submittedTxs.Transactions)
				must.NoError(ctx, err)
			}
		}()

		// drive received consensus values into cvpool
		s.wg.Add(1)
		go func() {
			defer s.wg.Done()
			for {
				sender, basis, ok := s.Multiplex.ReceiveConsensusBasis(s.ctx)
				if !ok {
					return
				}
				_ = sender
				if !protoutil.IsValidBasis(basis) {
					liblog.Errorf(ctx, "received cv at height %d is not valid, dropping",
						basis.Value.Proposal_header.Block_height)
					continue
				}
				s.CVPool.CVPoolAdd(s.ctx, *basis, basis.Value)
			}
		}()
	}()

	return Control{
		system:       s,
		conn:         schedConn,
		genesisReady: s.genesisReady,
	}
}
