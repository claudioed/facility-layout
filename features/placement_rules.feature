Feature: Placement rules
  A PlacementRule declares which LocationTypes are legal in which Zones. It
  is the mechanism that stops "ambient product in the frozen zone" —
  enforced once, at slot-registration time, naming the exact rule violated,
  rather than re-checked by every caller.

  Background:
    Given an empty warehouse map
    And a LocationType "PalletRack" with capacity 1200 kg and 2.4 m3
    And a LocationType "Shelf" with capacity 60 kg and 0.4 m3
    And a registered Site "WH1"

  @bdd
  Scenario: A location type denied by temperature class is rejected
    Given a registered Zone "STOR"/"FRZ" in Site "WH1" with temperature class "Frozen"
    And a registered Aisle "A02" in Zone "WH1-STOR-FRZ" with sequence hint 2
    And a PlacementRule "RULE-FRZ-NO-SHELF" denying "Shelf" where temperature class is "Frozen"
    When I register the LocationSlot "WH1-STOR-FRZ-A02-01-01-A" of type "Shelf"
    Then the response status is 422
    And the problem detail type is "placement-rule-violated"
    And the problem detail names the rule "RULE-FRZ-NO-SHELF"

  @bdd
  Scenario: A location type outside a zone's allow-list is rejected
    Given a registered Zone "STOR"/"HAZ" in Site "WH1" with temperature class "Ambient"
    And a registered Aisle "A01" in Zone "WH1-STOR-HAZ" with sequence hint 1
    And a PlacementRule "RULE-HAZ-ONLY-RACK" allowing "PalletRack" where zone code is "HAZ"
    When I register the LocationSlot "WH1-STOR-HAZ-A01-01-01-A" of type "Shelf"
    Then the response status is 422
    And the problem detail type is "placement-rule-violated"
    And the problem detail names the rule "RULE-HAZ-ONLY-RACK"

  @bdd
  Scenario: The allow-listed location type is accepted in the same zone
    Given a registered Zone "STOR"/"HAZ" in Site "WH1" with temperature class "Ambient"
    And a registered Aisle "A01" in Zone "WH1-STOR-HAZ" with sequence hint 1
    And a PlacementRule "RULE-HAZ-ONLY-RACK" allowing "PalletRack" where zone code is "HAZ"
    When I register the LocationSlot "WH1-STOR-HAZ-A01-01-01-A" of type "PalletRack"
    Then the response status is 201
    And the LocationSlot response reports type "PalletRack" with status "Active"

  @bdd
  Scenario: A zone no rule matches accepts anything
    Given a registered Zone "STOR"/"AMB" in Site "WH1" with temperature class "Ambient"
    And a registered Aisle "A07" in Zone "WH1-STOR-AMB" with sequence hint 7
    And a PlacementRule "RULE-FRZ-NO-SHELF" denying "Shelf" where temperature class is "Frozen"
    When I register the LocationSlot "WH1-STOR-AMB-A07-03-02-B" of type "Shelf"
    Then the response status is 201
