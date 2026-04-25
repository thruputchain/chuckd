package coordinator

import (
	"github.com/groundhogchain/lib-go/liblog"
)

func (s *System) Stop() {
	s.stopOnce.Do(func() {
		liblog.Infof(s.ctx, "coordinator stopping")

		s.signalGenesisIsReady()

		// outside network: discovery, dogwood, catchup, repair
		if s.outsideSrv != nil {
			liblog.Infof(s.ctx, "coordinator stopping outside network")
			s.outsideSrv.Stop()
		}
		if s.Dogwood != nil {
			s.Dogwood.Stop()
		}
		if s.Catchup != nil {
			s.Catchup.Stop()
		}
		if s.RepairService != nil {
			s.RepairService.Stop()
		}
		if s.Discovery != nil {
			s.Discovery.Stop()
		}
		if s.RepairService != nil {
			s.RepairService.Stop()
		}
		if s.UnvettedSubmitter != nil {
			s.UnvettedSubmitter.Stop()
		}
		if s.Wallet != nil {
			s.Wallet.Stop()
		}

		// inside network: scheduler calling coord
		if s.insideSrv != nil {
			liblog.Infof(s.ctx, "coordinator stopping inside network")
			s.insideSrv.Stop()
		}
		if s.CoordScheduler != nil {
			s.CoordScheduler.Stop()
		}

		// scp
		if s.SCPBroker != nil {
			s.SCPBroker.Stop()
		}
		if s.scpController != nil {
			s.scpController.Close()
		}
		if s.scpHandler != nil {
			s.scpHandler.Stop()
		}
		if s.scpdKill != nil {
			liblog.Infof(s.ctx, "coordinator killing scpd")
			s.scpdKill()
		}
		if s.SchedConn != nil {
			liblog.Infof(s.ctx, "coordinator closing scheduler connection")
			s.SchedConn.Close()
		}

		// XXX: if we are going to implement graceful stop, it should not be context-based,
		// so that interdependent services can be stopped in the correct order.
		if s.cancel != nil {
			s.cancel()
		}

		s.wg.Wait()

		liblog.Infof(s.ctx, "coordinator stopped")
	})
}

func (s *System) Close() {
	s.Stop()
}
