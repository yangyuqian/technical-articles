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

![Portfolio(Legacy) UI - Detail Page - NA](https://github.com/yangyuqian/technical-articles/blob/master/images/portfolio-legacy-detail-page-na.png)

In Transactional, NA shows the inventory matching the targeting criteria of placement. Transactional-NA = Displacing Guaranteed Ads + Displacing Preemptible Ads + Displacing Non-Paying Preemptible Ads + Unbooked Inventory(within the scope of targeting criteria), and it will ignore the budget settings on placement, and competing ads will be taken into consideration.

NA = 0 because it meet the end of its schedule:

![Transactional UI - NA - meet schedule](https://github.com/yangyuqian/technical-articles/blob/master/images/transactional-na-completed.png)

It seems ads won't be delivered due to price = 0:

![Transactional UI - NA - price 0](https://github.com/yangyuqian/technical-articles/blob/master/images/transactional-na-price-0.png)

Normally serving ads:

![Transactional UI - NA - normal serving](https://github.com/yangyuqian/technical-articles/blob/master/images/transactional-na-serving.png)


### Adjusted Capacity

Adjusted Capacity is the abbreviation of Adjusted Unconstraint Gross Avail(Adjusted UGA), which takes the inventory guranteed to resellers into consideration.

In Portfolio, Adjusted Capacity = PUGA (Gross Capacity) - Inventories granted to resellers with priority of "Hard Guaranteed without passback"

### Gross Avail

In Portfolio, there are Portfolio Unconstraint Gross Avail(PUGA).

In Transactional, there are TUGA(Transactional Unconstraint Gross Avail) and TCGA(Transactional Contraint Gross Avail).

Unconstraint Gross Avail(UGA) = Adjusted Capacity + Inventories granted to resellers with priority of "Hard Guaranteed without passback"

### Forecasted to Deliver

Forecasted to Deliver is the amount of ads will be delivered by the end of schedule, and it may be less than Booked Imps due to settings of exclusivity, capacity and competition. It's collected by simulating ad request to forecasting engine.

In Portfolio, Self Forecasted to Deliver = SUM(ads will be delivered of a portfolio)

Targeting | Portfolio v.s Transactional|Comment
----------|----------------------------|--------------------
boarder | Portfolio Self Forecasted to Deliver <= SUM(Transactional Impressions) | ?
narrower | Portfolio Self Forecasted to Deliver = SUM(Transactional Impressions) | Portfolio FTD is simulated by really impressions on those placement
uncomparable | Portfolio Self Forecasted to Deliver <=> SUM(Transactional Impressions) | ?

i.e.

![Portfolio FTD - targeting](https://github.com/yangyuqian/technical-articles/blob/master/images/portfolio-ftd.png)

### Competing Intensity

Competing Intensity shows "How much is my competitors targeting on my targeted inventories".


![Transactional - Competing Ads - Competing Intensity](https://github.com/yangyuqian/technical-articles/blob/master/images/transactional-competing-intensity.png)

![Portfolio - Competing Ads - Competing Intensity](https://github.com/yangyuqian/technical-articles/blob/master/images/portfolio-competing-intensity.png)

### FFDR

Forecast Final Delivery Rate(FFDR) is the forecasted delivery rate of budget of a placement.

FFDR = Forecasted to Delivered Budget * 100% / Total Budget

Total Budget:

Biz Case | Total Budget | DB column
---------|--------------|----------
Sponsor without volume cap or estimated impressions | if forecasted delivered impressions > 0, then FFDR = 100%, else 0% | ?
Sponsor with volume cap | volume cap | OLTP.ad_tree_node.event_goal
Sponsor with estimated impressions | estimated impressions | OLTP.placement.estimated_impressions
Normal Ad | Budget | if currency goal is set, then budget will be OLTP.ad_tree_node.currency_goal, and the currency goal should be transformed into impressions; If impression goal is set, budget means OLTP.ad_tree_node.event_goal

### OSI

On Schedule Indicator how fast the placement is delivered based on its pacing curve.

OSI > 100% shows ad is over delivered, and < 100% means under delivery, =100% means it exactly matches the pacing curve

OSI = Delivered Budget to Date * 100% / Budget Booked to be Delivered on this Date

### Straight-Line OSI

### Trend

### Consumed Impressions

### Transactional IMP

### Displacing

### SELF

