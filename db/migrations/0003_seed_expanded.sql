-- +goose Up
-- +goose StatementBegin

-- ─────────────────────────────────────────────────────────────────────
-- Additional destinations researched May 2026 — 30 new entries.
-- Source verification: J&K Tourism, Indiahikes, Bikat, Trip-related blogs.
-- ─────────────────────────────────────────────────────────────────────

INSERT INTO destinations
  (name, name_urdu, slug, region_id, district, tagline, uniqueness,
   location, altitude_m, best_months, season_type, rating, review_count,
   distance_from_srinagar_km, entry_fee_inr, network_coverage, practical, permits,
   is_published, is_featured)
VALUES
  ('Nigeen Lake', 'نگین جھیل', 'nigeen-lake',
   (SELECT id FROM regions WHERE slug='central'), 'Srinagar',
   'The quieter, more exclusive sibling of Dal Lake.',
   'Cleaner water, fewer houseboats, real peace for serious sunrise photography.',
   ST_GeogFromText('POINT(74.8419 34.1339)'), 1583, ARRAY[4,5,6,7,8,9,10], 'year-round', 4.7, 612, 4, 0,
   '{"jio":"good","airtel":"good","bsnl":"good"}', '{"atm":true,"fuelKm":1,"toilet":"clean","drone":false}',
   ARRAY[]::TEXT[], true, false),

  ('Mughal Gardens', NULL, 'mughal-gardens',
   (SELECT id FROM regions WHERE slug='central'), 'Srinagar',
   'Three terraced Mughal gardens overlooking Dal Lake.',
   'Shalimar (1619), Nishat (1633), Chashme Shahi (1632). Mughal landscape architecture.',
   ST_GeogFromText('POINT(74.8833 34.1167)'), 1587, ARRAY[4,5,6,7,8,9,10], 'summer', 4.6, 1789, 11, 24,
   '{"jio":"good","airtel":"good","bsnl":"good"}', '{"atm":true,"fuelKm":2,"toilet":"clean","drone":false}',
   ARRAY[]::TEXT[], true, false),

  ('Shankaracharya Temple', NULL, 'shankaracharya-temple',
   (SELECT id FROM regions WHERE slug='central'), 'Srinagar',
   '9th-century Shiva temple atop Zabarwan hill — best sunset over Dal.',
   '250 steps up. Visited by Adi Shankaracharya. Phones restricted in inner sanctum.',
   ST_GeogFromText('POINT(74.8430 34.0837)'), 1884, ARRAY[3,4,5,6,7,8,9,10,11], 'year-round', 4.7, 856, 6, 0,
   '{"jio":"good","airtel":"good","bsnl":"patchy"}', '{"atm":false,"fuelKm":4,"toilet":"basic","drone":false}',
   ARRAY[]::TEXT[], true, false),

  ('Pari Mahal', NULL, 'pari-mahal',
   (SELECT id FROM regions WHERE slug='central'), 'Srinagar',
   'Seven-terraced 17th-century pavilion above Chashme Shahi.',
   'Built by Dara Shikoh as a Sufi school. Floodlit at night.',
   ST_GeogFromText('POINT(74.8540 34.0848)'), 1740, ARRAY[4,5,6,7,8,9,10], 'summer', 4.6, 412, 9, 24,
   '{"jio":"good","airtel":"good","bsnl":"patchy"}', '{"atm":false,"fuelKm":3,"toilet":"basic","drone":false}',
   ARRAY[]::TEXT[], true, false),

  ('Jamia Masjid Srinagar', NULL, 'jamia-masjid-srinagar',
   (SELECT id FROM regions WHERE slug='central'), 'Srinagar',
   '14th-century mosque with 378 deodar pillars.',
   'Indo-Saracenic architecture. Avoid Fridays 12-2 PM (Jumma).',
   ST_GeogFromText('POINT(74.8123 34.0944)'), 1585, ARRAY[1,2,3,4,5,6,7,8,9,10,11,12], 'year-round', 4.7, 689, 4, 0,
   '{"jio":"good","airtel":"good","bsnl":"good"}', '{"atm":true,"fuelKm":1,"toilet":"basic","drone":false}',
   ARRAY[]::TEXT[], true, false),

  ('Hari Parbat', NULL, 'hari-parbat',
   (SELECT id FROM regions WHERE slug='central'), 'Srinagar',
   'Mughal fort + Sufi shrine + Sharika temple on a single sacred hill.',
   'The only hill where Muslim, Hindu and Sikh shrines coexist.',
   ST_GeogFromText('POINT(74.8085 34.1011)'), 1763, ARRAY[3,4,5,6,7,8,9,10], 'summer', 4.5, 287, 5, 0,
   '{"jio":"good","airtel":"good","bsnl":"patchy"}', '{"atm":false,"fuelKm":2,"toilet":"basic","drone":false}',
   ARRAY[]::TEXT[], true, false),

  ('Dachigam National Park', NULL, 'dachigam-national-park',
   (SELECT id FROM regions WHERE slug='central'), 'Srinagar',
   'Last habitat of the Hangul (Kashmir Stag). 141 km².',
   'Estimated ~250 Hangul remaining. Hiking trails through pine forest.',
   ST_GeogFromText('POINT(74.9000 34.1500)'), 1700, ARRAY[4,5,6,7,8,9,10,11,12], 'year-round', 4.6, 234, 22, 300,
   '{"jio":"patchy","airtel":"patchy","bsnl":"patchy"}', '{"atm":false,"fuelKm":10,"toilet":"basic","drone":false}',
   ARRAY['wildlife'], true, false),

  ('Hokersar Wetland', NULL, 'hokersar-wetland',
   (SELECT id FROM regions WHERE slug='central'), 'Srinagar',
   'Ramsar-listed bird sanctuary — winter migration spectacle.',
   '13.7 km² wetland. ~2 million migratory birds in winter.',
   ST_GeogFromText('POINT(74.7100 34.0931)'), 1584, ARRAY[11,12,1,2], 'winter', 4.4, 168, 14, 100,
   '{"jio":"good","airtel":"good","bsnl":"patchy"}', '{"atm":false,"fuelKm":5,"toilet":"basic","drone":false}',
   ARRAY['wildlife'], true, false),

  ('Aru Valley', 'آرو وادی', 'aru-valley',
   (SELECT id FROM regions WHERE slug='south'), 'Anantnag',
   '11 km north of Pahalgam — quieter meadow + trailhead for major treks.',
   'Where serious trekkers leave the road. Trailhead for Kolahoi and Tarsar-Marsar.',
   ST_GeogFromText('POINT(75.2625 34.0958)'), 2408, ARRAY[5,6,7,8,9,10], 'summer', 4.7, 521, 106, 0,
   '{"jio":"patchy","airtel":"patchy","bsnl":"none"}', '{"atm":false,"fuelKm":11,"toilet":"basic","drone":false}',
   ARRAY[]::TEXT[], true, false),

  ('Betaab Valley', NULL, 'betaab-valley',
   (SELECT id FROM regions WHERE slug='south'), 'Anantnag',
   'The 1983 Bollywood namesake — Lidder valley meadow framed by pines.',
   'Named after the Sunny Deol film. Easy day-trip from Pahalgam.',
   ST_GeogFromText('POINT(75.2603 34.0344)'), 2300, ARRAY[4,5,6,7,8,9,10], 'summer', 4.5, 1102, 102, 50,
   '{"jio":"good","airtel":"patchy","bsnl":"none"}', '{"atm":false,"fuelKm":7,"toilet":"basic","drone":false}',
   ARRAY[]::TEXT[], true, false),

  ('Chandanwari', NULL, 'chandanwari',
   (SELECT id FROM regions WHERE slug='south'), 'Anantnag',
   'Amarnath Yatra starting point. Year-round snow bridge over the Lidder.',
   'July-August: yatris head uphill to the cave. Snow bridge survives mild summers.',
   ST_GeogFromText('POINT(75.4317 34.0539)'), 2895, ARRAY[6,7,8,9], 'summer', 4.5, 423, 117, 0,
   '{"jio":"patchy","airtel":"none","bsnl":"patchy"}', '{"atm":false,"fuelKm":17,"toilet":"basic","drone":false}',
   ARRAY['amarnath-yatra'], true, false),

  ('Amarnath Cave', NULL, 'amarnath-cave',
   (SELECT id FROM regions WHERE slug='south'), 'Anantnag',
   'Sacred ice shivling at 3,888m. July-August only.',
   'Naturally-forming ice stalagmite. Two routes: Baltal or Chandanwari.',
   ST_GeogFromText('POINT(75.5006 34.2150)'), 3888, ARRAY[7,8], 'summer', 4.9, 2156, 145, 220,
   '{"jio":"none","airtel":"none","bsnl":"none"}', '{"atm":false,"fuelKm":30,"toilet":"none","drone":false}',
   ARRAY['amarnath-yatra'], true, false),

  ('Martand Sun Temple', NULL, 'martand-sun-temple',
   (SELECT id FROM regions WHERE slug='south'), 'Anantnag',
   'Ruins of an 8th-century sun temple — one of India''s oldest.',
   'Built by Lalitaditya Muktapida (725-756 CE). Kashmiri-style stone architecture.',
   ST_GeogFromText('POINT(75.2231 33.7459)'), 1600, ARRAY[3,4,5,6,7,8,9,10,11], 'year-round', 4.6, 489, 67, 25,
   '{"jio":"good","airtel":"good","bsnl":"patchy"}', '{"atm":true,"fuelKm":4,"toilet":"basic","drone":false}',
   ARRAY[]::TEXT[], true, false),

  ('Achabal Garden', NULL, 'achabal-garden',
   (SELECT id FROM regions WHERE slug='south'), 'Anantnag',
   'Mughal water garden with a freshwater spring as its source.',
   'Designed by Nur Jahan in 1620. Achabal Nag spring feeds the terraces.',
   ST_GeogFromText('POINT(75.2300 33.6900)'), 1670, ARRAY[4,5,6,7,8,9,10], 'summer', 4.5, 178, 60, 24,
   '{"jio":"good","airtel":"good","bsnl":"patchy"}', '{"atm":true,"fuelKm":2,"toilet":"basic","drone":false}',
   ARRAY[]::TEXT[], true, false),

  ('Verinag', NULL, 'verinag',
   (SELECT id FROM regions WHERE slug='south'), 'Anantnag',
   'Source of the Jhelum river — octagonal Mughal-era spring.',
   '24-metre deep blue spring, never dry. Jahangir built the embankment in 1620.',
   ST_GeogFromText('POINT(75.2500 33.5300)'), 1875, ARRAY[4,5,6,7,8,9,10], 'summer', 4.5, 234, 78, 24,
   '{"jio":"good","airtel":"patchy","bsnl":"patchy"}', '{"atm":false,"fuelKm":3,"toilet":"basic","drone":false}',
   ARRAY[]::TEXT[], true, false),

  ('Daksum', NULL, 'daksum',
   (SELECT id FROM regions WHERE slug='south'), 'Anantnag',
   'Pine-forested valley en route to Sinthan Top — a walker''s paradise.',
   '85 km from Srinagar. Bregwoo stream cuts through dense fir forest.',
   ST_GeogFromText('POINT(75.4500 33.5667)'), 2438, ARRAY[4,5,6,7,8,9,10], 'summer', 4.6, 156, 85, 0,
   '{"jio":"patchy","airtel":"patchy","bsnl":"none"}', '{"atm":false,"fuelKm":20,"toilet":"basic","drone":false}',
   ARRAY[]::TEXT[], true, false),

  ('Sinthan Top', NULL, 'sinthan-top',
   (SELECT id FROM regions WHERE slug='south'), 'Anantnag',
   'Mountain pass connecting Anantnag to Kishtwar — snow year-round.',
   '3,748 m road pass. Snow lingers into mid-summer.',
   ST_GeogFromText('POINT(75.5500 33.5167)'), 3748, ARRAY[6,7,8,9,10], 'summer', 4.6, 312, 130, 0,
   '{"jio":"patchy","airtel":"none","bsnl":"none"}', '{"atm":false,"fuelKm":35,"toilet":"none","drone":false}',
   ARRAY[]::TEXT[], true, false),

  ('Aharbal Waterfall', NULL, 'aharbal-waterfall',
   (SELECT id FROM regions WHERE slug='south'), 'Kulgam',
   'The "Niagara of Kashmir" — 25m cascade in the Pir Panjal.',
   'Vishav stream drops over basalt. Trailhead for Kounsarnag treks.',
   ST_GeogFromText('POINT(74.7500 33.6333)'), 2266, ARRAY[4,5,6,7,8,9,10], 'summer', 4.6, 467, 80, 30,
   '{"jio":"patchy","airtel":"patchy","bsnl":"none"}', '{"atm":false,"fuelKm":12,"toilet":"basic","drone":false}',
   ARRAY[]::TEXT[], true, false),

  ('Tulian Lake', NULL, 'tulian-lake',
   (SELECT id FROM regions WHERE slug='south'), 'Anantnag',
   'High-altitude alpine lake above Pahalgam — frozen most of the year.',
   'Trek from Pahalgam via Baisaran. Stays frozen until July.',
   ST_GeogFromText('POINT(75.4333 34.0167)'), 3684, ARRAY[7,8,9], 'summer', 4.7, 178, 105, 0,
   '{"jio":"none","airtel":"none","bsnl":"none"}', '{"atm":false,"fuelKm":12,"toilet":"none","drone":false}',
   ARRAY[]::TEXT[], true, false),

  ('Thajiwas Glacier', NULL, 'thajiwas-glacier',
   (SELECT id FROM regions WHERE slug='central'), 'Ganderbal',
   'Short 3 km uphill from Sonamarg — Kashmir''s most accessible glacier.',
   'Pony ride or walk. Sledging year-round.',
   ST_GeogFromText('POINT(75.2833 34.3167)'), 2900, ARRAY[5,6,7,8,9,10], 'summer', 4.5, 712, 83, 0,
   '{"jio":"patchy","airtel":"none","bsnl":"none"}', '{"atm":false,"fuelKm":6,"toilet":"basic","drone":false}',
   ARRAY[]::TEXT[], true, false),

  ('Naranag', NULL, 'naranag',
   (SELECT id FROM regions WHERE slug='central'), 'Ganderbal',
   'Trailhead for Naranag-Gangbal trek + 8th-century temple ruins.',
   'Two clusters of Shaiva temples from the Karkota dynasty.',
   ST_GeogFromText('POINT(74.9450 34.3350)'), 2200, ARRAY[5,6,7,8,9,10], 'summer', 4.6, 312, 50, 0,
   '{"jio":"patchy","airtel":"patchy","bsnl":"none"}', '{"atm":false,"fuelKm":18,"toilet":"basic","drone":false}',
   ARRAY[]::TEXT[], true, false),

  ('Manasbal Lake', NULL, 'manasbal-lake',
   (SELECT id FROM regions WHERE slug='central'), 'Ganderbal',
   '"Supreme gem of all Kashmiri lakes" — quiet alternative to Dal & Wular.',
   'Deepest lake in Kashmir at 13m. Lotus blooms in July-August.',
   ST_GeogFromText('POINT(74.6750 34.2403)'), 1583, ARRAY[4,5,6,7,8,9,10], 'summer', 4.6, 234, 32, 0,
   '{"jio":"good","airtel":"patchy","bsnl":"patchy"}', '{"atm":false,"fuelKm":5,"toilet":"basic","drone":false}',
   ARRAY[]::TEXT[], true, false),

  ('Kheer Bhawani', NULL, 'kheer-bhawani',
   (SELECT id FROM regions WHERE slug='central'), 'Ganderbal',
   'Sacred to Kashmiri Pandits — spring water changes colour with the times.',
   'Hexagonal spring temple. Major mela on Jyeshtha Ashtami (May-June).',
   ST_GeogFromText('POINT(74.7333 34.2167)'), 1580, ARRAY[4,5,6,7,8,9,10], 'year-round', 4.7, 367, 27, 0,
   '{"jio":"good","airtel":"good","bsnl":"good"}', '{"atm":true,"fuelKm":2,"toilet":"basic","drone":false}',
   ARRAY[]::TEXT[], true, false),

  ('Yusmarg', NULL, 'yusmarg',
   (SELECT id FROM regions WHERE slug='central'), 'Budgam',
   '"Meadow of Jesus" — quiet pine valley with Doodhganga river.',
   'Far less commercialised than Pahalgam. Trailhead for Nilnag Lake.',
   ST_GeogFromText('POINT(74.6667 33.8333)'), 2396, ARRAY[4,5,6,7,8,9,10], 'summer', 4.6, 312, 47, 0,
   '{"jio":"patchy","airtel":"patchy","bsnl":"patchy"}', '{"atm":false,"fuelKm":18,"toilet":"basic","drone":false}',
   ARRAY[]::TEXT[], true, false),

  ('Doodhpathri', NULL, 'doodhpathri',
   (SELECT id FROM regions WHERE slug='central'), 'Budgam',
   '"Valley of Milk" — meadow named for the milky-white Shaliganga river.',
   '40 km from Srinagar. Trailhead for Doodhpathri-Tosamaidan trek.',
   ST_GeogFromText('POINT(74.5419 33.8654)'), 2730, ARRAY[4,5,6,7,8,9,10], 'summer', 4.6, 387, 42, 50,
   '{"jio":"patchy","airtel":"patchy","bsnl":"none"}', '{"atm":false,"fuelKm":22,"toilet":"basic","drone":false}',
   ARRAY[]::TEXT[], true, false),

  ('Tosamaidan', NULL, 'tosamaidan',
   (SELECT id FROM regions WHERE slug='central'), 'Budgam',
   'High-altitude meadow above Doodhpathri.',
   'De-notified army artillery range; opened to civilians in 2014.',
   ST_GeogFromText('POINT(74.5500 33.7833)'), 3000, ARRAY[6,7,8,9], 'summer', 4.7, 123, 70, 0,
   '{"jio":"none","airtel":"none","bsnl":"none"}', '{"atm":false,"fuelKm":30,"toilet":"none","drone":false}',
   ARRAY[]::TEXT[], true, false),

  ('Charar-e-Sharief', NULL, 'charar-e-sharief',
   (SELECT id FROM regions WHERE slug='central'), 'Budgam',
   'Shrine of Sheikh Noor-ud-din Wali — the patron saint of Kashmir.',
   'Built 14th century, burnt 1995, rebuilt. "Alamdar-e-Kashmir".',
   ST_GeogFromText('POINT(74.7500 33.8333)'), 2100, ARRAY[3,4,5,6,7,8,9,10,11], 'year-round', 4.7, 289, 25, 0,
   '{"jio":"good","airtel":"good","bsnl":"patchy"}', '{"atm":true,"fuelKm":1,"toilet":"basic","drone":false}',
   ARRAY[]::TEXT[], true, false),

  ('Wular Lake', NULL, 'wular-lake',
   (SELECT id FROM regions WHERE slug='north'), 'Bandipora',
   'One of Asia''s largest freshwater lakes — 200 km².',
   'Fed by the Jhelum. Lotus blooms in summer. Critical bird flyway.',
   ST_GeogFromText('POINT(74.5667 34.3500)'), 1580, ARRAY[4,5,6,7,8,9,10], 'summer', 4.4, 312, 65, 0,
   '{"jio":"good","airtel":"patchy","bsnl":"patchy"}', '{"atm":true,"fuelKm":4,"toilet":"basic","drone":false}',
   ARRAY[]::TEXT[], true, false),

  ('Bangus Valley', NULL, 'bangus-valley',
   (SELECT id FROM regions WHERE slug='north'), 'Kupwara',
   '"Mini Switzerland" — 300 km² high-altitude meadow at 3,000m.',
   'Bordered by Shamasbari range. Largely untouched.',
   ST_GeogFromText('POINT(73.9333 34.4500)'), 3000, ARRAY[6,7,8,9], 'summer', 4.8, 156, 130, 0,
   '{"jio":"none","airtel":"none","bsnl":"none"}', '{"atm":false,"fuelKm":45,"toilet":"none","drone":false}',
   ARRAY['ILP'], true, false),

  ('Lolab Valley', 'لولاب وادی', 'lolab-valley',
   (SELECT id FROM regions WHERE slug='north'), 'Kupwara',
   '"Wadi-e-Lolab" — fruit orchards, springs, and rice fields.',
   'Three sub-valleys. Rich walnut + apple orchards. Kalaroos Caves nearby.',
   ST_GeogFromText('POINT(74.3000 34.5333)'), 1592, ARRAY[5,6,7,8,9,10], 'summer', 4.6, 234, 120, 0,
   '{"jio":"patchy","airtel":"patchy","bsnl":"patchy"}', '{"atm":true,"fuelKm":8,"toilet":"basic","drone":false}',
   ARRAY['ILP'], true, false),

  ('Patnitop', NULL, 'patnitop',
   (SELECT id FROM regions WHERE slug='central'), 'Udhampur',
   'Hill station at 2,024 m — pine forests, summer escape from Jammu heat.',
   'On NH-44 between Jammu and Srinagar. Cool summers, snow winters.',
   ST_GeogFromText('POINT(75.3270 33.0581)'), 2024, ARRAY[4,5,6,9,10,11,12,1,2], 'year-round', 4.4, 567, 222, 0,
   '{"jio":"good","airtel":"good","bsnl":"patchy"}', '{"atm":true,"fuelKm":3,"toilet":"clean","drone":false}',
   ARRAY[]::TEXT[], true, false),

  ('Vaishno Devi Katra', NULL, 'vaishno-devi',
   (SELECT id FROM regions WHERE slug='central'), 'Reasi',
   'One of Hinduism''s most venerated pilgrimages — 13 km mountain trek.',
   'Holy cave at 1,560m. 12 km trek from Katra base. 10+ million pilgrims/year.',
   ST_GeogFromText('POINT(74.9492 33.0306)'), 1560, ARRAY[3,4,5,6,9,10,11], 'year-round', 4.8, 4567, 285, 0,
   '{"jio":"good","airtel":"good","bsnl":"good"}', '{"atm":true,"fuelKm":0,"toilet":"clean","drone":false}',
   ARRAY[]::TEXT[], true, false),

  ('Bhaderwah', NULL, 'bhaderwah',
   (SELECT id FROM regions WHERE slug='central'), 'Doda',
   '"Mini Kashmir" of Doda — 71 km from Vaishno Devi.',
   'Hidden behind Padri Pass from Patnitop. Pristine, low tourist count.',
   ST_GeogFromText('POINT(75.7156 32.9856)'), 1613, ARRAY[4,5,6,7,8,9,10], 'summer', 4.7, 312, 196, 0,
   '{"jio":"good","airtel":"patchy","bsnl":"patchy"}', '{"atm":true,"fuelKm":2,"toilet":"basic","drone":false}',
   ARRAY[]::TEXT[], true, false),

  ('Kishtwar', NULL, 'kishtwar',
   (SELECT id FROM regions WHERE slug='central'), 'Kishtwar',
   'Saffron belt + 425 km² national park + the Chenab river canyon.',
   'Kishtwar NP is one of India''s most remote. Saffron rivals Pampore.',
   ST_GeogFromText('POINT(75.7700 33.3128)'), 1638, ARRAY[4,5,6,7,8,9,10], 'summer', 4.5, 156, 217, 0,
   '{"jio":"good","airtel":"patchy","bsnl":"patchy"}', '{"atm":true,"fuelKm":2,"toilet":"basic","drone":false}',
   ARRAY['wildlife'], true, false),

  ('Drass', NULL, 'drass',
   (SELECT id FROM regions WHERE slug='north'), 'Kargil',
   'Second-coldest inhabited place in the world after Oymyakon.',
   'Recorded -60°C in 1995. Tiger Hill and Tololing nearby.',
   ST_GeogFromText('POINT(75.7600 34.4500)'), 3230, ARRAY[6,7,8,9,10], 'summer', 4.6, 387, 144, 0,
   '{"jio":"patchy","airtel":"none","bsnl":"patchy"}', '{"atm":true,"fuelKm":1,"toilet":"basic","drone":false}',
   ARRAY[]::TEXT[], true, false)
ON CONFLICT (slug) DO NOTHING;

-- Categories for new destinations
INSERT INTO destination_categories (destination_id, category_id)
SELECT d.id, c.id FROM destinations d, categories c WHERE
  (d.slug = 'nigeen-lake' AND c.slug IN ('hidden-gems','nature')) OR
  (d.slug = 'mughal-gardens' AND c.slug IN ('popular','cultural','nature')) OR
  (d.slug = 'shankaracharya-temple' AND c.slug IN ('spiritual','cultural')) OR
  (d.slug = 'pari-mahal' AND c.slug IN ('cultural','hidden-gems')) OR
  (d.slug = 'jamia-masjid-srinagar' AND c.slug IN ('spiritual','cultural')) OR
  (d.slug = 'hari-parbat' AND c.slug IN ('spiritual','cultural','hidden-gems')) OR
  (d.slug = 'dachigam-national-park' AND c.slug IN ('nature','hidden-gems')) OR
  (d.slug = 'hokersar-wetland' AND c.slug IN ('nature','hidden-gems')) OR
  (d.slug = 'aru-valley' AND c.slug IN ('nature','adventure')) OR
  (d.slug = 'betaab-valley' AND c.slug IN ('popular','nature')) OR
  (d.slug = 'chandanwari' AND c.slug IN ('spiritual','adventure')) OR
  (d.slug = 'amarnath-cave' AND c.slug IN ('spiritual','adventure')) OR
  (d.slug = 'martand-sun-temple' AND c.slug IN ('cultural','spiritual','hidden-gems')) OR
  (d.slug = 'achabal-garden' AND c.slug IN ('cultural','hidden-gems')) OR
  (d.slug = 'verinag' AND c.slug IN ('cultural','hidden-gems')) OR
  (d.slug = 'daksum' AND c.slug IN ('hidden-gems','nature')) OR
  (d.slug = 'sinthan-top' AND c.slug IN ('adventure','hidden-gems')) OR
  (d.slug = 'aharbal-waterfall' AND c.slug IN ('nature','hidden-gems')) OR
  (d.slug = 'tulian-lake' AND c.slug IN ('adventure','hidden-gems')) OR
  (d.slug = 'thajiwas-glacier' AND c.slug IN ('adventure','nature')) OR
  (d.slug = 'naranag' AND c.slug IN ('hidden-gems','cultural','adventure')) OR
  (d.slug = 'manasbal-lake' AND c.slug IN ('hidden-gems','nature')) OR
  (d.slug = 'kheer-bhawani' AND c.slug IN ('spiritual','cultural')) OR
  (d.slug = 'yusmarg' AND c.slug IN ('hidden-gems','nature')) OR
  (d.slug = 'doodhpathri' AND c.slug IN ('hidden-gems','nature')) OR
  (d.slug = 'tosamaidan' AND c.slug IN ('hidden-gems','adventure')) OR
  (d.slug = 'charar-e-sharief' AND c.slug IN ('spiritual','cultural')) OR
  (d.slug = 'wular-lake' AND c.slug IN ('nature','hidden-gems')) OR
  (d.slug = 'bangus-valley' AND c.slug IN ('hidden-gems','nature','adventure')) OR
  (d.slug = 'lolab-valley' AND c.slug IN ('hidden-gems','nature','cultural')) OR
  (d.slug = 'patnitop' AND c.slug IN ('popular','nature')) OR
  (d.slug = 'vaishno-devi' AND c.slug IN ('spiritual','popular')) OR
  (d.slug = 'bhaderwah' AND c.slug IN ('hidden-gems','nature')) OR
  (d.slug = 'kishtwar' AND c.slug IN ('hidden-gems','nature','adventure')) OR
  (d.slug = 'drass' AND c.slug IN ('adventure','cultural','hidden-gems'))
ON CONFLICT DO NOTHING;

-- ─────────────────────────────────────────────────────────────────────
-- Additional treks (12 new) — Kolahoi, Warwan, Pir Panjal Lakes,
-- Nafran, Gurez Lakes, Bangus, Lolab, Doodhpathri, Tulian, Mahadev,
-- Aharbal-Kounsarnag, Amarnath Yatra.
-- ─────────────────────────────────────────────────────────────────────

INSERT INTO treks
  (slug, name, destination_id, difficulty, trek_type, duration_days, distance_km,
   max_altitude_m, start_point, end_point, best_months, ams_risk, status, closure_reason,
   tagline, uniqueness, rating, review_count, guide_available, guide_price_inr, permits, is_published)
VALUES
  ('kolahoi-glacier', 'Kolahoi Glacier',
   (SELECT id FROM destinations WHERE slug='aru-valley'),
   'moderate', 'glacier', 5, 26, 3840, 'Aru Valley', 'Aru Valley', ARRAY[6,7,8,9], true, 'closed',
   'Opens June when meadows are clear',
   'The "Goddess of Light" — Kolahoi peak and its glacier.',
   'Source of Lidder river. Lower-altitude alternative to KGL.',
   4.7, 489, true, 12000, ARRAY[]::TEXT[], true),

  ('warwan-valley', 'Warwan Valley',
   (SELECT id FROM destinations WHERE slug='sonamarg'),
   'hard', 'valley', 8, 110, 4500, 'Panikhar (Sonamarg-Kishtwar)', 'Lehinvan (Anantnag)', ARRAY[7,8], true, 'closed',
   'Snow-bound except July-August',
   'Kashmir''s most exquisite trek — 30 km hidden Himalayan valley.',
   '13 traditional Kashmiri villages. Highest difficulty trek in J&K.',
   4.9, 178, true, 28000, ARRAY[]::TEXT[], true),

  ('pir-panjal-lakes', 'Pir Panjal Lakes',
   (SELECT id FROM destinations WHERE slug='aharbal-waterfall'),
   'hard', 'alpine_lake', 9, 85, 4200, 'Aharbal (Kulgam)', 'Yusmarg (Budgam)', ARRAY[7,8,9], true, 'closed',
   'Long snow season — July to early October',
   'The grandest trek in J&K — chain of 7+ alpine lakes in Pir Panjal.',
   'Covers Kounsarnag (biggest), Sukhsar, Sundersar. Less crowded than KGL.',
   4.8, 145, true, 26000, ARRAY[]::TEXT[], true),

  ('nafran-valley', 'Nafran Valley',
   (SELECT id FROM destinations WHERE slug='sonamarg'),
   'moderate', 'valley', 6, 42, 3810, 'Sonamarg', 'Sonamarg', ARRAY[6,7,8,9], true, 'closed',
   'Snow-bound September onwards',
   'Hidden valley above Sonamarg — wildflowers, no crowds.',
   'Off the standard tourist path. Glaciated meadows, sheep herders'' pastures.',
   4.6, 89, true, 14000, ARRAY[]::TEXT[], true),

  ('gurez-lakes', 'Gurez Patalwan Lakes',
   (SELECT id FROM destinations WHERE slug='gurez-valley'),
   'hard', 'alpine_lake', 5, 38, 4100, 'Dawar (Gurez)', 'Dawar', ARRAY[7,8], true, 'closed',
   'Razdan Pass closes mid-Oct',
   'Twin lakes Patalwan Sar near the LoC — rare permit-only trek.',
   'Trekking along India-Pakistan border. Permit from Bandipora DC.',
   4.9, 67, true, 16000, ARRAY['ILP'], true),

  ('bangus-valley-trek', 'Bangus Valley Trek',
   (SELECT id FROM destinations WHERE slug='bangus-valley'),
   'moderate', 'meadow', 4, 32, 3500, 'Reshwari (Kupwara)', 'Reshwari', ARRAY[6,7,8,9], false, 'closed',
   'Snow returns October',
   'Walking the 300 km² "Mini Switzerland" meadow.',
   'Bodh Bangus (large) + Lokut Bangus (small). Camping under stars.',
   4.7, 56, true, 11000, ARRAY['ILP'], true),

  ('lolab-valley-trek', 'Lolab Valley Trek',
   (SELECT id FROM destinations WHERE slug='lolab-valley'),
   'easy', 'valley', 3, 22, 2400, 'Sogam (Kupwara)', 'Lalpora', ARRAY[5,6,7,8,9,10], false, 'open',
   NULL,
   'Easy 3-day walk through walnut orchards and rice fields.',
   'Forest walks + village stays. Suitable for first-time trekkers.',
   4.5, 89, true, 7500, ARRAY['ILP'], true),

  ('doodhpathri-tosamaidan', 'Doodhpathri-Tosamaidan',
   (SELECT id FROM destinations WHERE slug='doodhpathri'),
   'moderate', 'meadow', 3, 22, 3300, 'Doodhpathri', 'Tosamaidan', ARRAY[6,7,8,9,10], false, 'closed',
   'Snow on ridge October',
   'Two of Kashmir''s loveliest meadows linked by a ridgeline walk.',
   'Above tree line. Wildflowers carpet the ridge in July.',
   4.7, 67, true, 8500, ARRAY[]::TEXT[], true),

  ('tulian-lake-trek', 'Tulian Lake Day Trek',
   (SELECT id FROM destinations WHERE slug='pahalgam'),
   'moderate', 'alpine_lake', 2, 20, 3684, 'Baisaran (Pahalgam)', 'Baisaran', ARRAY[6,7,8,9], false, 'open',
   NULL,
   'A long day-hike to a high-altitude frozen lake.',
   'Frozen 9 months a year. Steep ascent past Baisaran meadow.',
   4.6, 234, true, 4500, ARRAY[]::TEXT[], true),

  ('mahadev-peak', 'Mahadev Peak',
   (SELECT id FROM destinations WHERE slug='shankaracharya-temple'),
   'moderate', 'pass', 2, 18, 3966, 'New Theed (Srinagar)', 'New Theed', ARRAY[4,5,6,7,8,9,10], false, 'open',
   NULL,
   'Highest peak in Srinagar district — sunrise summit, back by lunch.',
   'Zabarwan Range. Weekend trek from Srinagar. 360° valley view.',
   4.5, 167, true, 5000, ARRAY[]::TEXT[], true),

  ('aharbal-kounsarnag', 'Aharbal-Kounsarnag',
   (SELECT id FROM destinations WHERE slug='aharbal-waterfall'),
   'hard', 'alpine_lake', 7, 65, 4200, 'Aharbal', 'Aharbal', ARRAY[7,8,9], true, 'closed',
   'Snow on passes until July',
   'Pir Panjal''s biggest lake — 3 km wide Kounsarnag.',
   'Largest alpine lake in J&K. Steep, rocky, for experienced trekkers.',
   4.8, 78, true, 20000, ARRAY[]::TEXT[], true),

  ('amarnath-yatra', 'Amarnath Yatra (Pahalgam Route)',
   (SELECT id FROM destinations WHERE slug='chandanwari'),
   'hard', 'pilgrimage', 5, 48, 3888, 'Chandanwari (Pahalgam)', 'Baltal', ARRAY[7,8], true, 'closed',
   'Yatra opens early July, closes mid-Aug',
   'Sacred pilgrimage to the 3,888m ice shivling cave.',
   'Traditional Chandanwari-Sheshnag-Panchtarni-Amarnath route. Registration mandatory.',
   4.9, 3421, true, 22000, ARRAY['amarnath-yatra'], true)
ON CONFLICT (slug) DO NOTHING;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DELETE FROM treks WHERE slug IN (
  'kolahoi-glacier','warwan-valley','pir-panjal-lakes','nafran-valley','gurez-lakes',
  'bangus-valley-trek','lolab-valley-trek','doodhpathri-tosamaidan','tulian-lake-trek',
  'mahadev-peak','aharbal-kounsarnag','amarnath-yatra'
);

DELETE FROM destination_categories WHERE destination_id IN (
  SELECT id FROM destinations WHERE slug IN (
    'nigeen-lake','mughal-gardens','shankaracharya-temple','pari-mahal','jamia-masjid-srinagar',
    'hari-parbat','dachigam-national-park','hokersar-wetland','aru-valley','betaab-valley',
    'chandanwari','amarnath-cave','martand-sun-temple','achabal-garden','verinag','daksum',
    'sinthan-top','aharbal-waterfall','tulian-lake','thajiwas-glacier','naranag','manasbal-lake',
    'kheer-bhawani','yusmarg','doodhpathri','tosamaidan','charar-e-sharief','wular-lake',
    'bangus-valley','lolab-valley','patnitop','vaishno-devi','bhaderwah','kishtwar','drass'
  )
);

DELETE FROM destinations WHERE slug IN (
  'nigeen-lake','mughal-gardens','shankaracharya-temple','pari-mahal','jamia-masjid-srinagar',
  'hari-parbat','dachigam-national-park','hokersar-wetland','aru-valley','betaab-valley',
  'chandanwari','amarnath-cave','martand-sun-temple','achabal-garden','verinag','daksum',
  'sinthan-top','aharbal-waterfall','tulian-lake','thajiwas-glacier','naranag','manasbal-lake',
  'kheer-bhawani','yusmarg','doodhpathri','tosamaidan','charar-e-sharief','wular-lake',
  'bangus-valley','lolab-valley','patnitop','vaishno-devi','bhaderwah','kishtwar','drass'
);
-- +goose StatementEnd
