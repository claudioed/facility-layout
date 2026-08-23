Feature: Registering the warehouse structure
  A LocationSlot may only exist where its whole chain of custody resolves:
  the Site, Zone and Aisle its LocationCode names must all exist and all be
  Active. No orphan slots, ever.

  Background:
    Given an empty warehouse map
    And a LocationType "PalletRack" with capacity 1200 kg and 2.4 m3

  @bdd
  Scenario: Registering a full chain from site to slot succeeds
    When I register the Site "WH1" named "Fulfilment Centre One"
    Then the response status is 201
    And the response has a Location header pointing at "/sites/WH1"

    When I register the Zone "STOR"/"AMB" in Site "WH1" with temperature class "Ambient"
    Then the response status is 201
    And the response has a Location header pointing at "/zones/WH1-STOR-AMB"

    When I register the Aisle "A07" in Zone "WH1-STOR-AMB" with sequence hint 7 and direction "TwoWay"
    Then the response status is 201
    And the response has a Location header pointing at "/zones/WH1-STOR-AMB/aisles/A07"

    When I register the LocationSlot "WH1-STOR-AMB-A07-03-02-B" of type "PalletRack"
    Then the response status is 201
    And the response has a Location header pointing at "/locations/WH1-STOR-AMB-A07-03-02-B"
    And the LocationSlot response reports type "PalletRack" with status "Active"
    And the domain event "LocationSlotRegistered" was published

  @bdd
  Scenario: Registering a slot against an unknown aisle is rejected
    Given a registered Site "WH1"
    And a registered Zone "STOR"/"AMB" in Site "WH1" with temperature class "Ambient"
    And a registered Aisle "A07" in Zone "WH1-STOR-AMB" with sequence hint 7
    When I register the LocationSlot "WH1-STOR-AMB-A99-03-02-B" of type "PalletRack"
    Then the response status is 404
    And the problem detail type is "aisle-not-found"
    And no LocationSlot "WH1-STOR-AMB-A99-03-02-B" exists

  @bdd
  Scenario: Registering a slot against a decommissioned aisle is rejected
    Given a registered Site "WH1"
    And a registered Zone "STOR"/"AMB" in Site "WH1" with temperature class "Ambient"
    And a registered Aisle "A07" in Zone "WH1-STOR-AMB" with sequence hint 7
    And the Aisle "WH1-STOR-AMB-A07" has been decommissioned
    When I register the LocationSlot "WH1-STOR-AMB-A07-03-02-B" of type "PalletRack"
    Then the response status is 409
    And the problem detail type is "aisle-not-active"

  @bdd
  Scenario: Registering a zone against an unknown site is rejected
    When I register the Zone "STOR"/"AMB" in Site "WH9" with temperature class "Ambient"
    Then the response status is 404
    And the problem detail type is "site-not-found"

  @bdd
  Scenario: Registering a duplicate LocationCode is rejected
    Given a registered Site "WH1"
    And a registered Zone "STOR"/"AMB" in Site "WH1" with temperature class "Ambient"
    And a registered Aisle "A07" in Zone "WH1-STOR-AMB" with sequence hint 7
    And a registered LocationSlot "WH1-STOR-AMB-A07-03-02-B" of type "PalletRack"
    When I register the LocationSlot "WH1-STOR-AMB-A07-03-02-B" of type "PalletRack"
    Then the response status is 409
    And the problem detail type is "duplicate-location-code"

  @bdd
  Scenario: A decommissioned code cannot be resurrected by re-registration
    Given a registered Site "WH1"
    And a registered Zone "STOR"/"AMB" in Site "WH1" with temperature class "Ambient"
    And a registered Aisle "A07" in Zone "WH1-STOR-AMB" with sequence hint 7
    And a registered LocationSlot "WH1-STOR-AMB-A07-03-02-B" of type "PalletRack"
    When I decommission the LocationSlot "WH1-STOR-AMB-A07-03-02-B"
    Then the response status is 204
    And the domain event "LocationSlotDecommissioned" was published
    When I register the LocationSlot "WH1-STOR-AMB-A07-03-02-B" of type "PalletRack"
    Then the response status is 409
    And the problem detail type is "duplicate-location-code"
