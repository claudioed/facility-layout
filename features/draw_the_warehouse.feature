Feature: Drawing the warehouse
  The layout and grid endpoints exist to be RENDERED. They must come back
  pre-grouped and pre-ordered, so a frontend can paint a floor plan with no
  client-side joining, sorting, or layout maths.

  Background:
    Given an empty warehouse map
    And a LocationType "PalletRack" with capacity 1200 kg and 2.4 m3
    And a LocationType "Shelf" with capacity 60 kg and 0.4 m3
    And a registered Site "WH1"
    And a registered Zone "STOR"/"AMB" in Site "WH1" with temperature class "Ambient"
    And a registered Zone "RCV"/"AMB" in Site "WH1" with temperature class "Ambient"
    And a registered Aisle "A09" in Zone "WH1-STOR-AMB" with sequence hint 9
    And a registered Aisle "A07" in Zone "WH1-STOR-AMB" with sequence hint 7
    And a registered Aisle "D01" in Zone "WH1-RCV-AMB" with sequence hint 1
    And a registered LocationSlot "WH1-STOR-AMB-A07-03-02-B" of type "PalletRack"
    And a registered LocationSlot "WH1-STOR-AMB-A07-03-02-A" of type "PalletRack"
    And a registered LocationSlot "WH1-STOR-AMB-A07-03-01-A" of type "PalletRack"
    And a registered LocationSlot "WH1-STOR-AMB-A09-01-01-A" of type "Shelf"
    And a registered LocationSlot "WH1-RCV-AMB-D01-01-01-A" of type "Shelf"

  @bdd
  Scenario: The site layout comes back nested and pre-ordered for rendering
    When I request the layout of Site "WH1"
    Then the response status is 200
    And the layout reports 2 zones, 3 aisles and 5 slots
    And the layout zones are ordered "WH1-RCV-AMB,WH1-STOR-AMB"
    And the aisles of layout zone "WH1-STOR-AMB" are ordered "A07,A09"
    And the slots of layout aisle "WH1-STOR-AMB-A07" are ordered "WH1-STOR-AMB-A07-03-01-A,WH1-STOR-AMB-A07-03-02-A,WH1-STOR-AMB-A07-03-02-B"

  @bdd
  Scenario: The zone grid comes back as a matrix with null gaps
    When I request the grid of Zone "WH1-STOR-AMB"
    Then the response status is 200
    And the grid columns are ordered "A07/03,A09/01"
    And the grid levels are "01,02"
    And every grid row has one cell per column
    And the grid cell at level "02" column 0 holds positions "A,B"
    And the grid cell at level "02" column 1 is a gap

  @bdd
  Scenario: The layout renders as an SVG floor plan
    When I request the layout of Site "WH1" as SVG
    Then the response status is 200
    And the response content type is "image/svg+xml"
    And the response is a well-formed SVG document mentioning "WH1-STOR-AMB-A07-03-02-B"

  @bdd
  Scenario: Requesting the layout of an unknown site is rejected
    When I request the layout of Site "WH9"
    Then the response status is 404
    And the problem detail type is "site-not-found"
