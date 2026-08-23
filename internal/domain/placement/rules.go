package placement

import (
	"errors"
	"fmt"
)

// ErrPlacementRuleViolated is returned when a LocationType is not legal in
// the zone a slot is being registered into. The error text always names the
// specific rule that was violated.
var ErrPlacementRuleViolated = errors.New("location type violates a placement rule for this zone")

// RuleSet is the collection of PlacementRules applicable to a placement
// decision. Use cases load it from the PlacementRuleRepo and hand it to the
// LocationSlot constructor: an aggregate never reaches outside itself to
// query a repository.
type RuleSet []PlacementRule

// Check reports whether locationType may be placed in the zone described by
// attrs, per this rule set.
//
// Semantics, in evaluation order:
//
//  1. Any matching Deny rule naming this LocationType rejects it. Deny wins.
//  2. If any matching Allow rule exists for the zone at all, the zone is an
//     allow-list: the LocationType must be named by one of them.
//  3. Otherwise the zone is unconstrained and the placement is permitted.
func (rs RuleSet) Check(locationType string, attrs ZoneAttributes) error {
	allowListed := false
	allowed := false

	for _, rule := range rs {
		if !rule.Predicate().Matches(attrs) {
			continue
		}
		switch rule.Effect() {
		case Deny:
			if rule.LocationType() == locationType {
				return fmt.Errorf("%w: %s is denied in zone %s by rule [%s]",
					ErrPlacementRuleViolated, locationType, attrs.ZoneID, rule.Describe())
			}
		case Allow:
			allowListed = true
			if rule.LocationType() == locationType {
				allowed = true
			}
		}
	}

	if allowListed && !allowed {
		return fmt.Errorf("%w: zone %s allows only the location types named by its Allow rules, and %s is not among them (rules: %s)",
			ErrPlacementRuleViolated, attrs.ZoneID, locationType, rs.describeAllowRules(attrs))
	}
	return nil
}

func (rs RuleSet) describeAllowRules(attrs ZoneAttributes) string {
	described := ""
	for _, rule := range rs {
		if rule.Effect() != Allow || !rule.Predicate().Matches(attrs) {
			continue
		}
		if described != "" {
			described += "; "
		}
		described += "[" + rule.Describe() + "]"
	}
	return described
}
