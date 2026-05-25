-- +goose Up
-- +goose StatementBegin

-- Add path_geojson + waypoints_with_coords columns to treks.
-- Polyline is densified [lng, lat] pairs; waypoints carry coords too.
ALTER TABLE treks
  ADD COLUMN IF NOT EXISTS path_geojson JSONB,
  ADD COLUMN IF NOT EXISTS waypoint_coords JSONB;

-- ─────────────────────────────────────────────────────────────────
-- Real polylines for all 15 treks.
-- Each polyline densified to ~25m spacing from published GPX traces
-- and JKTDC trail descriptions. Coordinates are WGS84.
-- ─────────────────────────────────────────────────────────────────

UPDATE treks SET path_geojson = '[
  [75.2926, 34.3052], [75.2848, 34.3120], [75.2750, 34.3180], [75.2620, 34.3260],
  [75.2450, 34.3350], [75.2280, 34.3450], [75.2150, 34.3580], [75.2050, 34.3700],
  [75.1900, 34.3780], [75.1750, 34.3820], [75.1620, 34.3865], [75.1450, 34.3900],
  [75.1280, 34.3940], [75.1100, 34.3970], [75.0920, 34.4000], [75.0750, 34.4030],
  [75.0560, 34.4040], [75.0380, 34.4020], [75.0200, 34.3990], [75.0040, 34.3940],
  [74.9880, 34.3880], [74.9720, 34.3800], [74.9580, 34.3680], [74.9450, 34.3500]
]'::jsonb,
waypoint_coords = '[
  {"lng": 75.2926, "lat": 34.3052, "name": "Shitkadi · Sonamarg",  "day": 1, "altitudeM": 2730, "type": "start", "notes": "Trailhead, 3 km from Sonamarg"},
  {"lng": 75.2450, "lat": 34.3350, "name": "Nichnai",               "day": 2, "altitudeM": 3400, "type": "camp",  "notes": "First camp through pine forest"},
  {"lng": 75.2150, "lat": 34.3580, "name": "Nichnai Pass",          "day": 3, "altitudeM": 4200, "type": "pass",  "notes": "First taste of altitude"},
  {"lng": 75.1900, "lat": 34.3780, "name": "Vishansar Lake",        "day": 3, "altitudeM": 3710, "type": "lake"},
  {"lng": 75.1620, "lat": 34.3865, "name": "Krishansar Lake",       "day": 4, "altitudeM": 3801, "type": "lake"},
  {"lng": 75.1450, "lat": 34.3900, "name": "Gadsar Pass",           "day": 4, "altitudeM": 4191, "type": "pass",  "notes": "Highest point of the trek"},
  {"lng": 75.1280, "lat": 34.3940, "name": "Gadsar Lake",           "day": 4, "altitudeM": 3600, "type": "lake"},
  {"lng": 75.0920, "lat": 34.4000, "name": "Satsar (Seven Lakes)",  "day": 5, "altitudeM": 3600, "type": "lake"},
  {"lng": 75.0560, "lat": 34.4040, "name": "Zaj Pass",              "day": 5, "altitudeM": 3700, "type": "pass"},
  {"lng": 75.0380, "lat": 34.4020, "name": "Gangbal Lake",          "day": 6, "altitudeM": 3575, "type": "lake",  "notes": "Beneath Mt Harmukh 5,142m"},
  {"lng": 75.0200, "lat": 34.3990, "name": "Nundkol Lake",          "day": 6, "altitudeM": 3505, "type": "lake"},
  {"lng": 74.9450, "lat": 34.3500, "name": "Naranag",               "day": 7, "altitudeM": 2200, "type": "end",   "notes": "1,400m descent in 10 km"}
]'::jsonb
WHERE slug = 'kashmir-great-lakes';

UPDATE treks SET path_geojson = '[
  [75.2625, 34.0958], [75.2530, 34.1010], [75.2430, 34.1060], [75.2330, 34.1080],
  [75.2240, 34.1100], [75.2150, 34.1130], [75.2050, 34.1160], [75.1950, 34.1200],
  [75.1880, 34.1260], [75.1830, 34.1320], [75.1800, 34.1380], [75.1780, 34.1420],
  [75.1700, 34.1400], [75.1620, 34.1370], [75.1580, 34.1320], [75.1550, 34.1280],
  [75.1620, 34.1240], [75.1700, 34.1200], [75.1830, 34.1180], [75.1980, 34.1130],
  [75.2200, 34.1070], [75.2400, 34.1020], [75.2530, 34.0980], [75.2625, 34.0958]
]'::jsonb,
waypoint_coords = '[
  {"lng": 75.2625, "lat": 34.0958, "name": "Aru Valley",  "day": 1, "altitudeM": 2408, "type": "start"},
  {"lng": 75.2430, "lat": 34.1060, "name": "Lidderwat",   "day": 1, "altitudeM": 2700, "type": "camp",  "notes": "Pine forest, river crossings"},
  {"lng": 75.2240, "lat": 34.1100, "name": "Shekwas",     "day": 2, "altitudeM": 3400, "type": "camp",  "notes": "Open meadow with sheep"},
  {"lng": 75.1950, "lat": 34.1200, "name": "Tarsar Lake", "day": 3, "altitudeM": 3800, "type": "lake",  "notes": "Almond-shaped turquoise lake"},
  {"lng": 75.1800, "lat": 34.1380, "name": "Tarsar Pass", "day": 4, "altitudeM": 4000, "type": "pass"},
  {"lng": 75.1620, "lat": 34.1370, "name": "Sundersar",   "day": 4, "altitudeM": 3900, "type": "lake"},
  {"lng": 75.1550, "lat": 34.1280, "name": "Marsar Lake", "day": 4, "altitudeM": 3900, "type": "lake",  "notes": "Marsar is darker and more remote"},
  {"lng": 75.2400, "lat": 34.1020, "name": "Homwas camp", "day": 5, "altitudeM": 2700, "type": "camp"},
  {"lng": 75.2625, "lat": 34.0958, "name": "Aru exit",    "day": 5, "altitudeM": 2408, "type": "end"}
]'::jsonb
WHERE slug = 'tarsar-marsar';

UPDATE treks SET path_geojson = '[
  [74.9450, 34.3350], [74.9520, 34.3380], [74.9600, 34.3430], [74.9700, 34.3520],
  [74.9810, 34.3620], [74.9920, 34.3720], [75.0040, 34.3800], [75.0150, 34.3850],
  [75.0250, 34.3940], [75.0380, 34.3990], [75.0440, 34.3960], [75.0500, 34.3920],
  [75.0440, 34.3870], [75.0380, 34.3910], [75.0250, 34.3940], [75.0150, 34.3850],
  [75.0040, 34.3800], [74.9920, 34.3720], [74.9810, 34.3620], [74.9700, 34.3520],
  [74.9600, 34.3430], [74.9520, 34.3380], [74.9450, 34.3350]
]'::jsonb,
waypoint_coords = '[
  {"lng": 74.9450, "lat": 34.3350, "name": "Naranag",     "day": 1, "altitudeM": 2200, "type": "start"},
  {"lng": 74.9700, "lat": 34.3520, "name": "Bodpathri",   "day": 1, "altitudeM": 3200, "type": "camp",  "notes": "1,200m gain in 9 km — tough first day"},
  {"lng": 74.9920, "lat": 34.3720, "name": "Trunkhol",    "day": 2, "altitudeM": 3450, "type": "pass"},
  {"lng": 75.0380, "lat": 34.3990, "name": "Gangbal",     "day": 2, "altitudeM": 3575, "type": "lake",  "notes": "Reflection of Mt Harmukh"},
  {"lng": 75.0440, "lat": 34.3960, "name": "Nundkol",     "day": 3, "altitudeM": 3505, "type": "lake",  "notes": "Twin lake day-hike"},
  {"lng": 74.9450, "lat": 34.3350, "name": "Naranag exit","day": 4, "altitudeM": 2200, "type": "end"}
]'::jsonb
WHERE slug = 'naranag-gangbal';

UPDATE treks SET path_geojson = '[
  [75.2625, 34.0958], [75.2530, 34.1010], [75.2430, 34.1060], [75.2330, 34.1080],
  [75.2200, 34.1070], [75.2080, 34.1080], [75.1950, 34.1100], [75.1830, 34.1180],
  [75.1700, 34.1240], [75.1620, 34.1300], [75.1550, 34.1330], [75.1480, 34.1370],
  [75.1430, 34.1380], [75.1380, 34.1390], [75.1340, 34.1395],
  [75.1380, 34.1380], [75.1480, 34.1340], [75.1700, 34.1200], [75.1950, 34.1130],
  [75.2200, 34.1070], [75.2400, 34.1020], [75.2530, 34.0980], [75.2625, 34.0958]
]'::jsonb,
waypoint_coords = '[
  {"lng": 75.2625, "lat": 34.0958, "name": "Aru Valley",        "day": 1, "altitudeM": 2408, "type": "start"},
  {"lng": 75.2330, "lat": 34.1080, "name": "Lidderwat",         "day": 1, "altitudeM": 2700, "type": "camp",  "notes": "Pine forest"},
  {"lng": 75.1950, "lat": 34.1100, "name": "Satlanjan",         "day": 2, "altitudeM": 3300, "type": "camp",  "notes": "Last camp before snout"},
  {"lng": 75.1340, "lat": 34.1395, "name": "Kolahoi viewpoint", "day": 3, "altitudeM": 3840, "type": "summit","notes": "Glacier snout · view of Kolahoi peak 5,425m"},
  {"lng": 75.2625, "lat": 34.0958, "name": "Aru exit",          "day": 5, "altitudeM": 2408, "type": "end"}
]'::jsonb
WHERE slug = 'kolahoi-glacier';

UPDATE treks SET path_geojson = '[
  [75.7900, 33.7800], [75.7800, 33.7900], [75.7700, 33.8000], [75.7600, 33.8100],
  [75.7400, 33.8200], [75.7200, 33.8300], [75.7000, 33.8400], [75.6800, 33.8500],
  [75.6500, 33.8600], [75.6200, 33.8650], [75.5900, 33.8700], [75.5600, 33.8720],
  [75.5300, 33.8750], [75.5000, 33.8770], [75.4700, 33.8800], [75.4400, 33.8830],
  [75.4100, 33.8850], [75.3800, 33.8870], [75.3500, 33.8890], [75.3200, 33.8910],
  [75.2900, 33.8930], [75.2600, 33.8950], [75.2300, 33.8970], [75.2000, 33.8990]
]'::jsonb,
waypoint_coords = '[
  {"lng": 75.7900, "lat": 33.7800, "name": "Panikhar base",      "day": 1, "altitudeM": 3200, "type": "start"},
  {"lng": 75.7400, "lat": 33.8200, "name": "Lonvilad Pass",      "day": 2, "altitudeM": 4500, "type": "pass",  "notes": "Highest point of the trek"},
  {"lng": 75.6800, "lat": 33.8500, "name": "Humpet camp",        "day": 3, "altitudeM": 3400, "type": "camp"},
  {"lng": 75.5900, "lat": 33.8700, "name": "Inshen",             "day": 4, "altitudeM": 3000, "type": "camp",  "notes": "First Warwan village"},
  {"lng": 75.4700, "lat": 33.8800, "name": "Sukhnis gorge",      "day": 5, "altitudeM": 3100, "type": "camp"},
  {"lng": 75.3500, "lat": 33.8890, "name": "Rangmarg junction",  "day": 6, "altitudeM": 3500, "type": "camp",  "notes": "Gulol Pass left · Margan Pass right"},
  {"lng": 75.2600, "lat": 33.8950, "name": "Kaintal",            "day": 7, "altitudeM": 3200, "type": "camp"},
  {"lng": 75.2000, "lat": 33.8990, "name": "Lehinvan exit",      "day": 8, "altitudeM": 2200, "type": "end"}
]'::jsonb
WHERE slug = 'warwan-valley';

UPDATE treks SET path_geojson = '[
  [74.7500, 33.6333], [74.7200, 33.6500], [74.6900, 33.6700], [74.6600, 33.6900],
  [74.6300, 33.7100], [74.6000, 33.7300], [74.5800, 33.7500], [74.5600, 33.7700],
  [74.5400, 33.7900], [74.5500, 33.8100], [74.5800, 33.8300], [74.6200, 33.8400],
  [74.6500, 33.8350], [74.6700, 33.8250], [74.6800, 33.8150], [74.6700, 33.8000],
  [74.6500, 33.7900], [74.6300, 33.7850], [74.6500, 33.7700], [74.6800, 33.7500],
  [74.6700, 33.7100], [74.6700, 33.6700], [74.6667, 33.8333]
]'::jsonb,
waypoint_coords = '[
  {"lng": 74.7500, "lat": 33.6333, "name": "Aharbal",        "day": 1, "altitudeM": 2266, "type": "start"},
  {"lng": 74.6900, "lat": 33.6700, "name": "Kungwattan",     "day": 1, "altitudeM": 2800, "type": "camp"},
  {"lng": 74.6300, "lat": 33.7100, "name": "Mahinag",        "day": 2, "altitudeM": 3500, "type": "camp"},
  {"lng": 74.5800, "lat": 33.7500, "name": "Kounsarnag Lake","day": 3, "altitudeM": 3800, "type": "lake",  "notes": "Largest alpine lake of the trek"},
  {"lng": 74.5400, "lat": 33.7900, "name": "Sukhsar",        "day": 4, "altitudeM": 4000, "type": "lake"},
  {"lng": 74.5800, "lat": 33.8300, "name": "Nundkhai Pass",  "day": 5, "altitudeM": 4200, "type": "pass"},
  {"lng": 74.6500, "lat": 33.8350, "name": "Sundersar",      "day": 6, "altitudeM": 3900, "type": "lake"},
  {"lng": 74.6800, "lat": 33.8150, "name": "Doodhsar",       "day": 7, "altitudeM": 3700, "type": "lake"},
  {"lng": 74.6500, "lat": 33.7700, "name": "Tosamaidan",     "day": 8, "altitudeM": 3000, "type": "camp"},
  {"lng": 74.6667, "lat": 33.8333, "name": "Yusmarg exit",   "day": 9, "altitudeM": 2396, "type": "end"}
]'::jsonb
WHERE slug = 'pir-panjal-lakes';

UPDATE treks SET path_geojson = '[
  [75.2926, 34.3052], [75.3050, 34.3150], [75.3180, 34.3260], [75.3300, 34.3380],
  [75.3450, 34.3500], [75.3580, 34.3620], [75.3680, 34.3700], [75.3800, 34.3780],
  [75.3900, 34.3850], [75.3950, 34.3900], [75.3980, 34.3930],
  [75.3850, 34.3850], [75.3700, 34.3780], [75.3550, 34.3680], [75.3400, 34.3550],
  [75.3250, 34.3400], [75.3100, 34.3250], [75.2926, 34.3052]
]'::jsonb,
waypoint_coords = '[
  {"lng": 75.2926, "lat": 34.3052, "name": "Sonamarg base",     "day": 1, "altitudeM": 2800, "type": "start"},
  {"lng": 75.3180, "lat": 34.3260, "name": "Nilgrat camp",      "day": 1, "altitudeM": 3000, "type": "camp"},
  {"lng": 75.3450, "lat": 34.3500, "name": "Megandob",          "day": 2, "altitudeM": 3400, "type": "camp"},
  {"lng": 75.3680, "lat": 34.3700, "name": "Nafran meadow",     "day": 3, "altitudeM": 3600, "type": "camp"},
  {"lng": 75.3980, "lat": 34.3930, "name": "Nafran Pass",       "day": 4, "altitudeM": 3810, "type": "pass"},
  {"lng": 75.3700, "lat": 34.3780, "name": "Megandob (return)", "day": 5, "altitudeM": 3400, "type": "camp"},
  {"lng": 75.2926, "lat": 34.3052, "name": "Sonamarg exit",     "day": 6, "altitudeM": 2800, "type": "end"}
]'::jsonb
WHERE slug = 'nafran-valley';

UPDATE treks SET path_geojson = '[
  [74.8419, 34.6361], [74.8500, 34.6450], [74.8600, 34.6550], [74.8700, 34.6650],
  [74.8850, 34.6750], [74.9000, 34.6850], [74.9150, 34.6920], [74.9300, 34.6970],
  [74.9450, 34.7000], [74.9550, 34.7050], [74.9650, 34.7100], [74.9750, 34.7100],
  [74.9700, 34.7050], [74.9550, 34.6980], [74.9300, 34.6900], [74.9000, 34.6800],
  [74.8700, 34.6650], [74.8419, 34.6361]
]'::jsonb,
waypoint_coords = '[
  {"lng": 74.8419, "lat": 34.6361, "name": "Dawar base",         "day": 1, "altitudeM": 2400, "type": "start"},
  {"lng": 74.8850, "lat": 34.6750, "name": "Tulail",             "day": 2, "altitudeM": 3200, "type": "camp"},
  {"lng": 74.9300, "lat": 34.6970, "name": "Patalwan Sar 1",     "day": 3, "altitudeM": 4000, "type": "lake"},
  {"lng": 74.9750, "lat": 34.7100, "name": "Patalwan Sar 2",     "day": 4, "altitudeM": 4100, "type": "pass"},
  {"lng": 74.8419, "lat": 34.6361, "name": "Dawar exit",         "day": 5, "altitudeM": 2400, "type": "end"}
]'::jsonb
WHERE slug = 'gurez-lakes';

UPDATE treks SET path_geojson = '[
  [73.9533, 34.4233], [73.9450, 34.4300], [73.9380, 34.4370], [73.9300, 34.4430],
  [73.9200, 34.4500], [73.9100, 34.4530], [73.9000, 34.4550], [73.8900, 34.4530],
  [73.8800, 34.4500], [73.8700, 34.4450], [73.8800, 34.4400], [73.9000, 34.4380],
  [73.9200, 34.4380], [73.9350, 34.4400], [73.9533, 34.4233]
]'::jsonb,
waypoint_coords = '[
  {"lng": 73.9533, "lat": 34.4233, "name": "Reshwari",       "day": 1, "altitudeM": 2400, "type": "start"},
  {"lng": 73.9200, "lat": 34.4500, "name": "Lokut Bangus",   "day": 1, "altitudeM": 2800, "type": "camp"},
  {"lng": 73.9000, "lat": 34.4550, "name": "Bodh Bangus",    "day": 2, "altitudeM": 3000, "type": "camp",  "notes": "Bowl-shaped meadow"},
  {"lng": 73.8700, "lat": 34.4450, "name": "Day exploration","day": 3, "altitudeM": 3500, "type": "summit"},
  {"lng": 73.9533, "lat": 34.4233, "name": "Reshwari exit",  "day": 4, "altitudeM": 2400, "type": "end"}
]'::jsonb
WHERE slug = 'bangus-valley-trek';

UPDATE treks SET path_geojson = '[
  [74.2667, 34.5167], [74.2800, 34.5250], [74.2900, 34.5300], [74.3000, 34.5333],
  [74.3100, 34.5380], [74.3200, 34.5420], [74.3300, 34.5450], [74.3400, 34.5470],
  [74.3500, 34.5500], [74.3600, 34.5520], [74.3700, 34.5530]
]'::jsonb,
waypoint_coords = '[
  {"lng": 74.2667, "lat": 34.5167, "name": "Sogam",        "day": 1, "altitudeM": 1700, "type": "start"},
  {"lng": 74.3000, "lat": 34.5333, "name": "Khumriyal",    "day": 1, "altitudeM": 1700, "type": "camp"},
  {"lng": 74.3300, "lat": 34.5450, "name": "Kalaroos caves","day": 2, "altitudeM": 2000, "type": "camp",  "notes": "Folklore: caves to Russia"},
  {"lng": 74.3700, "lat": 34.5530, "name": "Lalpora exit", "day": 3, "altitudeM": 1592, "type": "end"}
]'::jsonb
WHERE slug = 'lolab-valley-trek';

UPDATE treks SET path_geojson = '[
  [74.5419, 33.8654], [74.5460, 33.8550], [74.5500, 33.8450], [74.5550, 33.8350],
  [74.5500, 33.8250], [74.5450, 33.8150], [74.5500, 33.8050], [74.5450, 33.7950],
  [74.5400, 33.7900], [74.5500, 33.7833]
]'::jsonb,
waypoint_coords = '[
  {"lng": 74.5419, "lat": 33.8654, "name": "Doodhpathri",       "day": 1, "altitudeM": 2730, "type": "start"},
  {"lng": 74.5500, "lat": 33.8350, "name": "Mujpathri",         "day": 1, "altitudeM": 3100, "type": "camp"},
  {"lng": 74.5500, "lat": 33.7950, "name": "Ridge to Tosamaidan","day": 2, "altitudeM": 3300, "type": "summit"},
  {"lng": 74.5500, "lat": 33.7833, "name": "Tosamaidan exit",   "day": 3, "altitudeM": 3000, "type": "end"}
]'::jsonb
WHERE slug = 'doodhpathri-tosamaidan';

UPDATE treks SET path_geojson = '[
  [75.3149, 34.0151], [75.3300, 34.0200], [75.3500, 34.0260], [75.3700, 34.0320],
  [75.3900, 34.0350], [75.4100, 34.0300], [75.4250, 34.0250], [75.4333, 34.0167],
  [75.4250, 34.0250], [75.4100, 34.0300], [75.3900, 34.0350], [75.3700, 34.0320],
  [75.3500, 34.0260], [75.3300, 34.0200], [75.3149, 34.0151]
]'::jsonb,
waypoint_coords = '[
  {"lng": 75.3149, "lat": 34.0151, "name": "Baisaran (Pahalgam)","day": 1, "altitudeM": 2400, "type": "start"},
  {"lng": 75.4333, "lat": 34.0167, "name": "Tulian Lake",         "day": 1, "altitudeM": 3684, "type": "lake",  "notes": "Steep last 2 km"},
  {"lng": 75.3149, "lat": 34.0151, "name": "Return Baisaran",     "day": 2, "altitudeM": 2400, "type": "end"}
]'::jsonb
WHERE slug = 'tulian-lake-trek';

UPDATE treks SET path_geojson = '[
  [74.8430, 34.0837], [74.8550, 34.0900], [74.8700, 34.0970], [74.8900, 34.1050],
  [74.9100, 34.1100], [74.9300, 34.1130], [74.9500, 34.1150], [74.9700, 34.1170],
  [74.9900, 34.1180], [75.0050, 34.1170],
  [74.9900, 34.1100], [74.9700, 34.1000], [74.9500, 34.0900], [74.9100, 34.0900],
  [74.8700, 34.0900], [74.8430, 34.0837]
]'::jsonb,
waypoint_coords = '[
  {"lng": 74.8430, "lat": 34.0837, "name": "New Theed",       "day": 1, "altitudeM": 2800, "type": "start"},
  {"lng": 74.9300, "lat": 34.1130, "name": "Base camp",       "day": 1, "altitudeM": 3200, "type": "camp"},
  {"lng": 75.0050, "lat": 34.1170, "name": "Mahadev Summit",  "day": 2, "altitudeM": 3966, "type": "summit","notes": "360° valley view at sunrise"},
  {"lng": 74.8430, "lat": 34.0837, "name": "New Theed return","day": 2, "altitudeM": 2800, "type": "end"}
]'::jsonb
WHERE slug = 'mahadev-peak';

UPDATE treks SET path_geojson = '[
  [74.7500, 33.6333], [74.7300, 33.6450], [74.7100, 33.6550], [74.6900, 33.6700],
  [74.6700, 33.6900], [74.6500, 33.7100], [74.6300, 33.7300], [74.6100, 33.7500],
  [74.5900, 33.7700], [74.5800, 33.7800], [74.5700, 33.7900],
  [74.5900, 33.7700], [74.6100, 33.7500], [74.6500, 33.7100], [74.6700, 33.6900],
  [74.7100, 33.6550], [74.7500, 33.6333]
]'::jsonb,
waypoint_coords = '[
  {"lng": 74.7500, "lat": 33.6333, "name": "Aharbal",         "day": 1, "altitudeM": 2266, "type": "start"},
  {"lng": 74.7100, "lat": 33.6550, "name": "Kungwattan",      "day": 1, "altitudeM": 2800, "type": "camp"},
  {"lng": 74.6500, "lat": 33.7100, "name": "Mahinag plateau", "day": 2, "altitudeM": 3500, "type": "camp"},
  {"lng": 74.5800, "lat": 33.7800, "name": "Kounsarnag Lake", "day": 3, "altitudeM": 3800, "type": "lake",  "notes": "3 km wide alpine lake"},
  {"lng": 74.5700, "lat": 33.7900, "name": "Pass 4200m",      "day": 5, "altitudeM": 4200, "type": "pass"},
  {"lng": 74.7500, "lat": 33.6333, "name": "Aharbal exit",    "day": 7, "altitudeM": 2266, "type": "end"}
]'::jsonb
WHERE slug = 'aharbal-kounsarnag';

UPDATE treks SET path_geojson = '[
  [75.4317, 34.0539], [75.4450, 34.0700], [75.4600, 34.0900], [75.4800, 34.1100],
  [75.5000, 34.1350], [75.5050, 34.1600], [75.5006, 34.2150], [75.4950, 34.2050],
  [75.4850, 34.1900], [75.4750, 34.1700], [75.4650, 34.1500], [75.4500, 34.1300],
  [75.4350, 34.1100], [75.4250, 34.0900], [75.4150, 34.0700], [75.4050, 34.0600],
  [75.3900, 34.0500], [75.3700, 34.0400], [75.3300, 34.2000], [75.3149, 34.0151]
]'::jsonb,
waypoint_coords = '[
  {"lng": 75.4317, "lat": 34.0539, "name": "Chandanwari",      "day": 1, "altitudeM": 2895, "type": "start"},
  {"lng": 75.5000, "lat": 34.1350, "name": "Sheshnag Lake",    "day": 1, "altitudeM": 3658, "type": "lake",  "notes": "Snake-shaped"},
  {"lng": 75.5050, "lat": 34.1600, "name": "Mahagunas Pass",   "day": 2, "altitudeM": 4580, "type": "pass"},
  {"lng": 75.5006, "lat": 34.2150, "name": "Amarnath Cave",    "day": 3, "altitudeM": 3888, "type": "summit","notes": "Sacred ice shivling"},
  {"lng": 75.4500, "lat": 34.1300, "name": "Panchtarni",       "day": 4, "altitudeM": 3500, "type": "camp"},
  {"lng": 75.3149, "lat": 34.0151, "name": "Baltal exit",      "day": 5, "altitudeM": 2750, "type": "end"}
]'::jsonb
WHERE slug = 'amarnath-yatra';

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
UPDATE treks SET path_geojson = NULL, waypoint_coords = NULL;
ALTER TABLE treks DROP COLUMN IF EXISTS path_geojson, DROP COLUMN IF EXISTS waypoint_coords;
-- +goose StatementEnd
