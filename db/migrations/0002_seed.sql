-- +goose Up
-- +goose StatementBegin

-- Regions
INSERT INTO regions (name, slug, description) VALUES
  ('North Kashmir',   'north',   'Bandipora, Baramulla, Kupwara — Gulmarg, Gurez, Bangus, Lolab.'),
  ('Central Kashmir', 'central', 'Srinagar, Ganderbal, Budgam — Dal Lake, Sonamarg, Yousmarg, Doodhpathri.'),
  ('South Kashmir',   'south',   'Anantnag, Pulwama, Kulgam, Shopian — Pahalgam, Aru, Betaab, Martand.');

-- Categories
INSERT INTO categories (name, slug, icon, color) VALUES
  ('Popular',      'popular',     'star',     '#E8893A'),
  ('Adventure',    'adventure',   'mountain', '#B23A2E'),
  ('Nature',       'nature',      'tree',     '#2D6A4F'),
  ('Cultural',     'cultural',    'theater',  '#C9A227'),
  ('Spiritual',    'spiritual',   'moon',     '#1F4788'),
  ('Hidden Gems',  'hidden-gems', 'gem',      '#8B4513');

-- Roads
INSERT INTO roads (name, slug, current_status, closure_reason) VALUES
  ('NH-44 Srinagar–Jammu',           'nh44',         'one-way',     'Maintenance Banihal–Qazigund, 11am–3pm'),
  ('NH-1D Srinagar–Leh (Zojila)',    'nh1d-zojila',  'open',        NULL),
  ('Mughal Road · Shopian–Poonch',   'mughal-road',  'open',        NULL),
  ('Razdan Pass · Bandipora–Gurez',  'razdan',       'closed',      'First snow — closed until spring 2027'),
  ('Sinthan Top Road',               'sinthan',      'open',        NULL);

-- Destinations
INSERT INTO destinations
  (name, name_urdu, name_hindi, slug, region_id, district, tagline, uniqueness,
   location, altitude_m, best_months, season_type, rating, review_count,
   distance_from_srinagar_km, entry_fee_inr, network_coverage, practical, permits,
   is_published, is_featured)
VALUES
  (
    'Gulmarg', 'گلمرگ', 'गुलमर्ग', 'gulmarg',
    (SELECT id FROM regions WHERE slug = 'north'), 'Baramulla',
    'Asia''s highest ski resort. World''s 2nd-highest gondola.',
    'The Gulmarg Gondola climbs to 4,200m at Apharwat Peak — only Mérida in Venezuela goes higher. 18-hole golf course in summer, deep powder in winter.',
    ST_GeogFromText('POINT(74.3805 34.0488)'), 2650,
    ARRAY[12,1,2,3,4,5,9,10], 'year-round', 4.8, 1247, 56, 0,
    '{"jio":"good","airtel":"good","bsnl":"patchy"}', '{"atm":true,"fuelKm":2,"toilet":"clean","drone":false}',
    ARRAY[]::TEXT[], true, true
  ),
  (
    'Dal Lake', 'ڈَل جھیل', 'डल झील', 'dal-lake',
    (SELECT id FROM regions WHERE slug = 'central'), 'Srinagar',
    'Srinagar''s 18 km² jewel — shikara, houseboats, sunrise floating markets.',
    'About 1,500 ornately-carved houseboats line the lake — a relic of British-era workarounds to the ban on foreign land ownership. Char Chinari island sits at the centre with four chinar trees.',
    ST_GeogFromText('POINT(74.8531 34.1218)'), 1583,
    ARRAY[4,5,6,7,8,9,10,12,1,2], 'year-round', 4.7, 2891, 0, 0,
    '{"jio":"good","airtel":"good","bsnl":"good"}', '{"atm":true,"fuelKm":1,"toilet":"clean","drone":false}',
    ARRAY[]::TEXT[], true, true
  ),
  (
    'Pahalgam', 'پہلگام', 'पहलगाम', 'pahalgam',
    (SELECT id FROM regions WHERE slug = 'south'), 'Anantnag',
    'Valley of Shepherds. Gateway to Amarnath, base for the great treks.',
    'The Lidder River cuts through pine and meadow here. From Chandanwari (12km north) the Amarnath Yatra begins.',
    ST_GeogFromText('POINT(75.3149 34.0151)'), 2130,
    ARRAY[4,5,6,7,8,9,10,11,12,1], 'year-round', 4.7, 1834, 95, 0,
    '{"jio":"good","airtel":"patchy","bsnl":"patchy"}', '{"atm":true,"fuelKm":0,"toilet":"clean","drone":false}',
    ARRAY[]::TEXT[], true, true
  ),
  (
    'Sonamarg', 'سونہ مرگ', 'सोनामर्ग', 'sonamarg',
    (SELECT id FROM regions WHERE slug = 'central'), 'Ganderbal',
    'Meadow of Gold. Gateway to Ladakh via Zojila — open May to October only.',
    'Thajiwas Glacier is 3 km from town. Zojila Pass (3,528m) east connects to Drass and Kargil — closes November to April.',
    ST_GeogFromText('POINT(75.2926 34.3032)'), 2800,
    ARRAY[5,6,7,8,9,10], 'summer', 4.6, 1102, 80, 0,
    '{"jio":"good","airtel":"patchy","bsnl":"patchy"}', '{"atm":true,"fuelKm":3,"toilet":"basic","drone":false}',
    ARRAY[]::TEXT[], true, true
  ),
  (
    'Gurez Valley', 'گریز وادی', 'गुरेज़ घाटी', 'gurez-valley',
    (SELECT id FROM regions WHERE slug = 'north'), 'Bandipora',
    'Remote valley along the Kishanganga. ILP required. Open June to September only.',
    'Habba Khatoon peak rises 4,142m above Dawar town, named after Kashmir''s 16th-century poet-queen.',
    ST_GeogFromText('POINT(74.8419 34.6361)'), 2400,
    ARRAY[6,7,8,9], 'summer', 4.9, 287, 125, 0,
    '{"jio":"patchy","airtel":"none","bsnl":"patchy"}', '{"atm":false,"fuelKm":30,"toilet":"basic","drone":false}',
    ARRAY['ILP'], true, false
  ),
  (
    'Indira Gandhi Tulip Garden', 'ٹیولپ گارڈن', 'ट्यूलिप गार्डन', 'tulip-garden',
    (SELECT id FROM regions WHERE slug = 'central'), 'Srinagar',
    'Asia''s largest tulip garden. 1.7 million blooms. Open late March to mid-April only.',
    '73 tulip varieties planted in terraces on the slopes of Zabarwan range, overlooking Dal Lake.',
    ST_GeogFromText('POINT(74.8589 34.0908)'), 1592,
    ARRAY[3,4], 'summer', 4.6, 1567, 9, 75,
    '{"jio":"good","airtel":"good","bsnl":"good"}', '{"atm":true,"fuelKm":1,"toilet":"clean","drone":false}',
    ARRAY[]::TEXT[], true, true
  ),
  (
    'Hazratbal Shrine', 'حضرتبل', 'हज़रतबल', 'hazratbal-shrine',
    (SELECT id FROM regions WHERE slug = 'central'), 'Srinagar',
    'Sacred shrine on Dal Lake, holds the Moe-e-Muqaddas relic.',
    'The white marble shrine is Kashmir''s most venerated Muslim site. Modest dress required.',
    ST_GeogFromText('POINT(74.8419 34.1267)'), 1585,
    ARRAY[1,2,3,4,5,6,7,8,9,10,11,12], 'year-round', 4.8, 943, 9, 0,
    '{"jio":"good","airtel":"good","bsnl":"good"}', '{"atm":true,"fuelKm":1,"toilet":"clean","drone":false}',
    ARRAY[]::TEXT[], true, false
  ),
  (
    'Aru Valley', 'آرو وادی', 'अरु घाटी', 'aru-valley',
    (SELECT id FROM regions WHERE slug = 'south'), 'Anantnag',
    '11 km north of Pahalgam — quieter meadow + the trailhead for Kolahoi & Tarsar-Marsar.',
    'Aru is where serious trekkers leave the road. Lidder river flowing through; trailhead for major south-valley treks.',
    ST_GeogFromText('POINT(75.2625 34.0958)'), 2408,
    ARRAY[5,6,7,8,9,10], 'summer', 4.7, 521, 106, 0,
    '{"jio":"patchy","airtel":"patchy","bsnl":"none"}', '{"atm":false,"fuelKm":11,"toilet":"basic","drone":false}',
    ARRAY[]::TEXT[], true, false
  );

-- Destination categories (m2m)
INSERT INTO destination_categories (destination_id, category_id)
SELECT d.id, c.id FROM destinations d, categories c WHERE
  (d.slug = 'gulmarg' AND c.slug IN ('popular','adventure')) OR
  (d.slug = 'dal-lake' AND c.slug IN ('popular','cultural')) OR
  (d.slug = 'pahalgam' AND c.slug IN ('popular','adventure','nature')) OR
  (d.slug = 'sonamarg' AND c.slug IN ('popular','adventure','nature')) OR
  (d.slug = 'gurez-valley' AND c.slug IN ('hidden-gems','nature')) OR
  (d.slug = 'tulip-garden' AND c.slug IN ('popular','nature','cultural')) OR
  (d.slug = 'hazratbal-shrine' AND c.slug IN ('spiritual','cultural')) OR
  (d.slug = 'aru-valley' AND c.slug IN ('nature','adventure'));

-- Activities (sample)
INSERT INTO destination_activities (destination_id, activity)
SELECT d.id, a FROM destinations d, UNNEST(ARRAY['Skiing','Gondola','Golf','Trekking','Photography']) a
WHERE d.slug = 'gulmarg';

-- Treks (subset)
INSERT INTO treks
  (slug, name, destination_id, difficulty, trek_type, duration_days, distance_km,
   max_altitude_m, start_point, end_point, best_months, ams_risk, status, closure_reason,
   tagline, uniqueness, rating, review_count, guide_available, guide_price_inr, is_published)
VALUES
  (
    'kashmir-great-lakes', 'Kashmir Great Lakes',
    (SELECT id FROM destinations WHERE slug='sonamarg'),
    'moderate', 'alpine_lake', 7, 72, 4191,
    'Shitkadi (Sonamarg)', 'Naranag',
    ARRAY[7,8,9], true, 'closed', 'Snow on passes — opens early July',
    'Seven alpine lakes in seven days. The classic Kashmir trek.',
    'Crosses four glacial passes and seven alpine lakes — Vishansar, Krishansar, Gadsar, Satsar, the twin Gangbal-Nundkol, and the smaller Yamsar.',
    4.9, 2103, true, 18000, true
  ),
  (
    'tarsar-marsar', 'Tarsar Marsar',
    (SELECT id FROM destinations WHERE slug='aru-valley'),
    'moderate', 'alpine_lake', 5, 48, 4000,
    'Aru Valley', 'Aru Valley',
    ARRAY[7,8,9], true, 'closed', 'Opens July when snow clears the pass',
    'Twin almond-shaped lakes. Quieter alternative to Great Lakes.',
    'Tarsar and Marsar are two almond-shaped alpine lakes that share a ridgeline.',
    4.8, 1421, true, 13000, true
  ),
  (
    'naranag-gangbal', 'Naranag–Gangbal',
    (SELECT id FROM destinations WHERE slug='sonamarg'),
    'moderate', 'alpine_lake', 4, 32, 3575,
    'Naranag', 'Naranag',
    ARRAY[6,7,8,9], true, 'closed', 'Opens June when snow melts',
    'Twin alpine lakes Gangbal and Nundkol beneath Mt Harmukh (5,142m).',
    'Mt Harmukh is sacred to Kashmiri Pandits — its reflection is said to be visible in Gangbal lake.',
    4.7, 712, true, 10000, true
  );

-- Providers (subset)
INSERT INTO providers (type, name, jktdc_reg_no, verified, base_location_text, languages,
                        rating, review_count, phone, whatsapp, capacity, amenities,
                        price_inr, price_unit, cancellation, description, years_hosting,
                        response_time_min)
VALUES
  ('houseboat', 'Heritage HB Princess of Kashmir', 'JKT/HB/12345', true, 'Dal Lake · Boulevard side',
   ARRAY['English','Hindi','Urdu','French'], 4.8, 287, '+919876543210', '+919876543210', 8,
   ARRAY['Wi-Fi','Hot water','Wazwan included','Pickup from SXR','Heated rooms'],
   6500, 'per-night', 'Free up to 7 days · 50% up to 48 hr · No refund <48 hr',
   'Deluxe-tier houseboat with 4 ensuite rooms and a central drawing room with intricate khatamband ceiling. Owner is a 4th-generation host.', 22, 12),
  ('shikara', 'Bashir Ahmad · Shikara No. 47', 'JKT/SK/3892', true, 'Ghat 7 · Boulevard',
   ARRAY['Hindi','Urdu','Kashmiri','Basic English'], 4.9, 412, '+919876543211', '+919876543211', 6,
   ARRAY['Sunrise tour','Floating market','Lotus garden','Tea served'],
   600, 'per-hour', 'Free cancellation up to 12 hr before',
   'Two-hour sunrise route via Char Chinari and the lotus garden.', 18, 5),
  ('guide', 'Mehboob Ali · IMF certified', 'JKT/TG/0921', true, 'Sonamarg',
   ARRAY['English','Hindi','Urdu','Ladakhi'], 4.9, 156, '+919876543212', '+919876543212', 8,
   ARRAY['All meals','Sleeping bag','Tents','Permits handled','Pony porter'],
   18000, 'per-trip', '50% refund up to 21 days · forfeit if <14 days',
   'IMF-certified trek leader. Specialises in Kashmir Great Lakes and Tarsar Marsar.', 11, 30);

-- Initial advisories
INSERT INTO advisories (severity, category, title, body, source, affected, confidence, effective_to)
VALUES
  ('critical', 'road', 'Razdan Pass closed — Gurez inaccessible',
   'First snow on Razdan Pass overnight. JKTDC has closed the road.',
   'JKTDC', 'Bandipora → Gurez Valley', 100, now() + INTERVAL '180 days'),
  ('warning', 'weather', 'Heavy snowfall expected — Gulmarg & Sonamarg',
   '25–40 cm of fresh snow expected over the next 48 hr.',
   'IMD', 'Gulmarg, Sonamarg, Zojila', 85, now() + INTERVAL '48 hours');

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DELETE FROM advisories;
DELETE FROM providers;
DELETE FROM treks;
DELETE FROM destination_activities;
DELETE FROM destination_categories;
DELETE FROM destinations;
DELETE FROM roads;
DELETE FROM categories;
DELETE FROM regions;
-- +goose StatementEnd
