package scpapi

import (
	"bytes"
	"sort"

	"github.com/groundhogchain/chuckd/util/protoutil"
	proto "github.com/groundhogchain/protocol-go"
)

/*
// MaximumQuorumNestingLevel is the maximum allowed nesting depth for quorum sets
// level = 0 when there is no nesting
const MaximumQuorumNestingLevel = 4

// QSetError represents errors that can occur during quorum set validation

	type QSetError struct {
		Message string
	}

	func (e *QSetError) Error() string {
		return e.Message
	}

// quorumSetSanityChecker validates quorum set configurations

	type quorumSetSanityChecker struct {
		extraChecks bool
		knownNodes  map[proto.ValidatorID]struct{}
		isSane      bool
		count       int
	}

	func newQuorumSetSanityChecker(qset *proto.SCPQuorumSet, extraChecks bool) (*quorumSetSanityChecker, error) {
		checker := &quorumSetSanityChecker{
			extraChecks: extraChecks,
			knownNodes:  make(map[proto.ValidatorID]struct{}),
			isSane:      true,
			count:       0,
		}

		var err error
		checker.isSane, err = checker.checkSanity(qset, 0)
		if err != nil {
			return nil, err
		}

		if checker.isSane && (checker.count < 1 || checker.count > 1000) {
			return nil, &QSetError{
				Message: "total number of nodes in a quorum must be within 1 and 1000",
			}
		}

		return checker, nil
	}

	func (c *quorumSetSanityChecker) checkSanity(qset *proto.SCPQuorumSet, depth uint32) (bool, error) {
		if depth > MaximumQuorumNestingLevel {
			return false, &QSetError{
				Message: "maximum quorum nesting level exceeded",
			}
		}

		if qset.Threshold < 1 {
			return false, &QSetError{
				Message: "threshold must be greater than 0",
			}
		}

		validators := qset.Validators
		innerSets := qset.Inner_sets

		totEntries := len(validators) + len(innerSets)
		vBlockingSize := totEntries + 1 - int(qset.Threshold)
		if vBlockingSize < 0 {
			vBlockingSize = 0
		}
		c.count += len(validators)

		if int(qset.Threshold) > totEntries {
			return false, &QSetError{
				Message: "threshold exceeds total number of entries",
			}
		}

		// threshold is within the proper range
		if c.extraChecks && int(qset.Threshold) < vBlockingSize {
			return false, &QSetError{
				Message: "threshold is lower than the v-blocking size (< 51%)",
			}
		}

		for _, node := range validators {
			if _, exists := c.knownNodes[protoutil.NodeToValidatorID(node)]; exists {
				return false, &QSetError{
					Message: "duplicate node found in quorum configuration",
				}
			}
			c.knownNodes[protoutil.NodeToValidatorID(node)] = struct{}{}
		}

		for i := range innerSets {
			_, err := c.checkSanity(&innerSets[i], depth+1)
			if err != nil {
				return false, err
			}
		}

		return true, nil
	}

// IsQSetSane validates whether a quorum set is sane

	func IsQSetSane(qset *proto.SCPQuorumSet, extraChecks bool) error {
		checker, err := newQuorumSetSanityChecker(qset, extraChecks)
		if err != nil {
			return err
		}
		if !checker.isSane {
			return &QSetError{Message: "quorum set is not sane"}
		}
		return nil
	}
*/

func normalizeQSetSimplify(qset *proto.SCPQuorumSet, idToRemove *proto.PublicKey) {
	// remove the specified node ID if provided
	if idToRemove != nil {
		originalLen := len(qset.Validators)
		validators := qset.Validators[:0]
		for _, node := range qset.Validators {
			if protoutil.NodeToValidatorID(node) != protoutil.NodeToValidatorID(*idToRemove) {
				validators = append(validators, node)
			}
		}
		qset.Validators = validators
		removedCount := originalLen - len(qset.Validators)
		if qset.Threshold > uint32(removedCount) {
			qset.Threshold -= uint32(removedCount)
		} else {
			qset.Threshold = 0
		}
	}

	// process inner sets
	unprocessed := make([]proto.SCPQuorumSet, len(qset.Inner_sets))
	copy(unprocessed, qset.Inner_sets)
	qset.Inner_sets = qset.Inner_sets[:0]

	for len(unprocessed) > 0 {
		innerSet := unprocessed[len(unprocessed)-1]
		unprocessed = unprocessed[:len(unprocessed)-1]

		normalizeQSetSimplify(&innerSet, idToRemove)

		if qset.Threshold == 1 && innerSet.Threshold == 1 {
			for _, innerInnerSet := range innerSet.Inner_sets {
				unprocessed = append(unprocessed, innerInnerSet)
			}
			for _, innerValidator := range innerSet.Validators {
				found := false
				for _, v := range qset.Validators {
					if protoutil.NodeToValidatorID(v) == protoutil.NodeToValidatorID(innerValidator) {
						found = true
						break
					}
				}
				if !found {
					qset.Validators = append(qset.Validators, innerValidator)
				}
			}
		} else {
			qset.Inner_sets = append(qset.Inner_sets, innerSet)
		}
	}

	// simplify quorum set if needed
	// (threshold=1, no validators, 1 inner set)
	if qset.Threshold == 1 && len(qset.Validators) == 0 && len(qset.Inner_sets) == 1 {
		innerSet := qset.Inner_sets[0]
		*qset = innerSet
	}
}

func normalizeQuorumSetReorder(qset *proto.SCPQuorumSet) {
	sortedPublicKeys(qset.Validators).Sort()
	for i := range qset.Inner_sets {
		normalizeQuorumSetReorder(&qset.Inner_sets[i])
	}
	sortedQSets(qset.Inner_sets).Sort()
}

// NormalizeQSet normalizes a quorum set by simplifying and reordering
func NormalizeQSet(qset *proto.SCPQuorumSet, idToRemove *proto.PublicKey) {
	normalizeQSetSimplify(qset, idToRemove)
	normalizeQuorumSetReorder(qset)
}

// Sort public keys
type sortedPublicKeys []proto.PublicKey

func (ks sortedPublicKeys) Len() int      { return len(ks) }
func (ks sortedPublicKeys) Swap(i, j int) { ks[i], ks[j] = ks[j], ks[i] }
func (ks sortedPublicKeys) Less(i, j int) bool {
	return comparePublicKeys(&ks[i], &ks[j]) < 0
}
func (ks sortedPublicKeys) Sort() { sort.Sort(ks) }

func comparePublicKeys(l, r *proto.PublicKey) int {
	left := protoutil.NodeToValidatorID(*l)
	right := protoutil.NodeToValidatorID(*r)
	return bytes.Compare(left[:], right[:])
}

// Sort quorum sets
type sortedQSets []proto.SCPQuorumSet

func (ks sortedQSets) Len() int      { return len(ks) }
func (ks sortedQSets) Swap(i, j int) { ks[i], ks[j] = ks[j], ks[i] }
func (ks sortedQSets) Less(i, j int) bool {
	return compareQSets(&ks[i], &ks[j]) < 0
}
func (ks sortedQSets) Sort() { sort.Sort(ks) }

func compareQSets(l, r *proto.SCPQuorumSet) int {
	// Compare by validators first - lexicographically (element by element)
	minLen := len(l.Validators)
	if len(r.Validators) < minLen {
		minLen = len(r.Validators)
	}

	for i := 0; i < minLen; i++ {
		if cmp := comparePublicKeys(&l.Validators[i], &r.Validators[i]); cmp != 0 {
			return cmp
		}
	}

	// If we've compared all common elements and they're equal, shorter vector is less than
	if len(l.Validators) != len(r.Validators) {
		if len(l.Validators) < len(r.Validators) {
			return -1
		}
		return 1
	}

	// Then compare by inner sets (lexicographically, element by element)
	minInnerLen := len(l.Inner_sets)
	if len(r.Inner_sets) < minInnerLen {
		minInnerLen = len(r.Inner_sets)
	}

	for i := 0; i < minInnerLen; i++ {
		if cmp := compareQSets(&l.Inner_sets[i], &r.Inner_sets[i]); cmp != 0 {
			return cmp
		}
	}

	// If we've compared all common elements and they're equal, shorter vector is less than
	if len(l.Inner_sets) != len(r.Inner_sets) {
		if len(l.Inner_sets) < len(r.Inner_sets) {
			return -1
		}
		return 1
	}

	// Finally compare by threshold
	if l.Threshold < r.Threshold {
		return -1
	} else if l.Threshold > r.Threshold {
		return 1
	}
	return 0
}
