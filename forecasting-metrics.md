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

- An evergreen(without budget) placement meets its end date, known as soft cutoff, then its Booked Impressions will be 0

- Ads can compete with each other through currency budget

Known Issues:

- Forecasting engine doesn't honor the daypart targeting

UI Parts:
Portfolio UI(New)

![Portfolio(New) UI - Landing Page - Booked Imps](https://github.com/yangyuqian/technical-articles/blob/master/images/portfolio-new-landing-page-booked.png)
![Portfolio(New) UI - Detail Page - Booked Imps](https://github.com/yangyuqian/technical-articles/blob/master/images/portfolio-new-detail-page-booked.png)

Portfolio UI(Legacy)

![Portfolio(Legacy) UI - Booked Imps](https://github.com/yangyuqian/technical-articles/blob/master/images/portfolio-legacy-summary-booked.png)

Transactional UI

![Transactional UI - Booked Imps](https://github.com/yangyuqian/technical-articles/blob/master/images/transactional-detail-page-booked.png)

Please see [Calculate Booked Impressions](http://wiki.dev.fwmrm.net/display/wq/How+to+calculate+Book+Imps), [Booked Impressions in Portfolio](http://wiki.dev.fwmrm.net/display/ForecastPortal/Portfolio+Metrics+---+Booked+Impression), [Booked Impressions in Transactional](http://wiki.dev.fwmrm.net/display/ForecastPortal/Transactional+Metrics+---+Booked+Impressions) for more details.

### Net Avail

In Portfolio, NA(Net Avail) indicates the amount of inventory that is available for sale, in another word, NA Ad opportunities of inventory segments inside the portfolio settings will be unfilled according to the forecasting result.

![Portfolio(New) UI - Landing Page - NA](https://github.com/yangyuqian/technical-articles/blob/master/images/portfolio-landing-page-na.png)

![Portfolio(New) UI - Detail Page - NA](https://github.com/yangyuqian/technical-articles/blob/master/images/portfolio-detail-page-na.png)

In Transactional, NA shows the inventory matching the targeting criteria of placement. Transactional-NA = Displacing Guaranteed Ads + Displacing Preemptible Ads + Displacing Non-Paying Preemptible Ads + Unbooked Inventory(within the scope of targeting criteria), and it will ignore the budget settings on placement, and competing ads will be taken into consideration.

NA = 0 because it meet the end of its schedule:

![Transactional UI - NA - meet schedule](https://github.com/yangyuqian/technical-articles/blob/master/images/transactional-na-completed.png)

It seems ads won't be delivered due to price = 0:

![Transactional UI - NA - price 0](https://github.com/yangyuqian/technical-articles/blob/master/images/transactional-na-price-0.png)

Normally serving ads:

![Transactional UI - NA - normal serving](https://github.com/yangyuqian/technical-articles/blob/master/images/transactional-na-serving.png)


### Adjusted Capacity

### Gross Avail

### Forecasted to Deliver

### Competing Intensity

### FFDR

### OSI

### Straight-Line OSI

### Trend

### Consumed Impressions

### Transactional IMP

### Displacing

### SELF

