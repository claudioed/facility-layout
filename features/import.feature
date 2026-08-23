Feature: Importing a facility layout
  Bulk import is how a real building's layout gets loaded once, reproducibly,
  from a CSV/JSON export — not typed in slot by slot. Validation is atomic
  per row: every row is processed, and the report says exactly which rows
  failed and why.

  Background:
    Given an empty warehouse map
    And a LocationType "PalletRack" with capacity 1200 kg and 2.4 m3
    And a LocationType "Shelf" with capacity 60 kg and 0.4 m3

  @bdd
  Scenario: Importing a clean layout creates the whole structure
    When I import the following facility layout rows:
      | siteCode | areaCode | zoneCode | temperatureClass | aisleCode | sequenceHint | bay | level | position | locationType |
      | WH1      | STOR     | AMB      | Ambient          | A07       | 7            | 03  | 01    | A        | PalletRack   |
      | WH1      | STOR     | AMB      | Ambient          | A07       | 7            | 03  | 02    | B        | PalletRack   |
      | WH1      | STOR     | FRZ      | Frozen           | A02       | 2            | 01  | 01    | A        | PalletRack   |
    Then the response status is 200
    And the import report says 3 submitted, 3 imported, 0 rejected
    And the domain event "FacilityLayoutImported" was published
    And the Site "WH1" exists
    And the Zone "WH1-STOR-FRZ" exists
    And the LocationSlot "WH1-STOR-AMB-A07-03-02-B" exists

  @bdd
  Scenario: Importing a mixed layout reports partial success per row
    Given a PlacementRule "RULE-FRZ-NO-SHELF" denying "Shelf" where temperature class is "Frozen"
    When I import the following facility layout rows:
      | siteCode | areaCode | zoneCode | temperatureClass | aisleCode | sequenceHint | bay | level | position | locationType |
      | WH1      | STOR     | AMB      | Ambient          | A07       | 7            | 03  | 01    | A        | PalletRack   |
      | WH1      | STOR     | AMB      | Ambient          | A07       | 7            | 03  | 02    | b        | PalletRack   |
      | WH1      | STOR     | AMB      | Ambient          | A07       | 7            | 03  | 03    | A        | Hovercraft   |
      | WH1      | STOR     | FRZ      | Frozen           | A02       | 2            | 01  | 01    | A        | Shelf        |
      | WH1      | STOR     | AMB      | Ambient          | A07       | 7            | 04  | 01    | A        | PalletRack   |
    Then the response status is 200
    And the import report says 5 submitted, 2 imported, 3 rejected
    And import row 1 was rejected mentioning "uppercase letters and digits"
    And import row 2 was rejected mentioning "location type not found"
    And import row 3 was rejected mentioning "RULE-FRZ-NO-SHELF"
    And import row 0 succeeded
    And import row 4 succeeded
    And the LocationSlot "WH1-STOR-AMB-A07-04-01-A" exists
