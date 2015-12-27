# Transactional v.s. Portfolio

[Transactional Forecasting](http://hub.freewheel.tv/display/MUG/MRM+Transactional+Forecasting) could just as easily be called Campaign Forecasts. They show delivery and inventory projections as they relate to a particular Placement. [Portfolio forecasts](http://hub.freewheel.tv/display/MUG/MRM+Portfolio+Forecasting) (sometimes called inventory forecasts) are projections on well-defined segments of inventory.

Basically, Transactional & Portfolio are implemented with the same forecasting logic in the engine. What makes the difference of Transactional and Portfolio is that the engine forecasts an existing placement, created by clients to serve ads, for Transactional but a non-existing placement, created by ui to simulate an existing placement, for Portfolio.

Transactional forecasting provides measurement on the ad delivery of some placement, and Portfolio forecasting shows the potential ad delivery of inventory combination in a non-existing placement.

## Metrics

[Forecasting Metrics](http://wiki.dev.fwmrm.net/display/ForecastPortal/Forecasting+Metrics)

Scenario Forecasting

### Booked Impressions

Competitor | Algorithm | AF Module
-------------|------------|----------
External Ad | GA of external ad * competing intensity | aggregator
Sponsor Ad | if Sponsor Competitor meet it's budget, then Booked Imps will be considered; if there are 1+ sponsor Ads inside a single slot, then UGA of this slot will be divided into equivalent Booked Imps of each Ad; if slot is sponsored by some ad(and no sponsor competitor), then Booked Imps = UGA; | aggregator or simulator
Exclusivity (normal) | just like sponsor competitors | aggregator(?)
Linked(Companion) Display | the Companion diplay Ad will be counted as the booked Impressions of the 'standalone' ad, which can not be served individually due to companion compatitors | (?)

Normal cases along with competitors:

- A evergreen/plc meets its end date, known as soft cutoff, then its Booked Impressions will be 0

- Ads can compete with each other through currency budget

Known Issues:

- Forecasting engine doesn't honor the daypart targeting

Please see [Calculate Booked Impressions](http://wiki.dev.fwmrm.net/display/wq/How+to+calculate+Book+Imps), [Booked Impressions in Portfolio](http://wiki.dev.fwmrm.net/display/ForecastPortal/Portfolio+Metrics+---+Booked+Impression), [Booked Impressions in Transactional](http://wiki.dev.fwmrm.net/display/ForecastPortal/Transactional+Metrics+---+Booked+Impressions) for more details.


