-- +goose Up
-- +goose StatementBegin

-- ─── Permits table ───────────────────────────────────────────
CREATE TABLE permits (
  id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  slug            TEXT UNIQUE NOT NULL,
  name            TEXT NOT NULL,
  required        TEXT NOT NULL,
  office          TEXT NOT NULL,
  processing_days TEXT NOT NULL,
  cost_inr        TEXT NOT NULL,
  validity        TEXT NOT NULL,
  status          TEXT NOT NULL,
  notes           TEXT,
  official_url    TEXT,
  created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

INSERT INTO permits (slug, name, required, office, processing_days, cost_inr, validity, status, notes, official_url) VALUES
  ('ilp',       'Inner Line Permit',
   'Gurez, Keran, Karnah, Tangdhar, Machil, Bangus',
   'DC Bandipora · DC Kupwara · or local SSP',
   '1–3 days', 'Free or nominal', 'Trip-specific', 'always',
   'Carry Aadhaar or Passport. Foreigners restricted in some areas.',
   'https://jkpolice.gov.in'),
  ('amarnath',  'Amarnath Yatra Registration',
   'Amarnath Cave',
   'SASB online + designated banks',
   '5–7 days', '₹220', 'Specific dates · Jul–Aug only', 'seasonal',
   'Medical certificate mandatory. Routes: Baltal (short/steep) or Chandanwari (long/traditional).',
   'https://jksasb.nic.in'),
  ('wildlife',  'Wildlife Permit',
   'Dachigam NP, Overa-Aru, Hokersar wetland',
   'J&K Wildlife Dept · Srinagar',
   'Same day', '₹300 Indian · ₹2,500 foreign', 'Per visit', 'open',
   'Camera fee extra. Dachigam upper area requires special permission.',
   'https://www.jkforest.com'),
  ('drone',     'Drone Permit',
   'All drone flying in J&K',
   'DGCA Digital Sky · MoCA',
   '7–15 days', '₹1,000+', 'Per location · per date', 'always',
   'Most J&K areas are no-fly. Apply at least 2 weeks ahead.',
   'https://digitalsky.dgca.gov.in'),
  ('frro',      'Foreign Tourist Registration',
   'Foreign nationals near LOC areas',
   'FRRO Srinagar',
   'Same day', 'Free', 'Duration of stay', 'always',
   'Required within 24 hr of arrival in restricted areas. Hotel can usually help.',
   'https://indianfrro.gov.in');

-- ─── Photo spots seed (a representative subset per destination) ─────
INSERT INTO photo_spots (destination_id, name, location, best_time, facing, tripod_recommended, drone_allowed, description)
SELECT id, 'Boulevard Road sunrise',                 ST_GeogFromText('POINT(74.8533 34.1232)'), 'sunrise',    'NE',  true,  false, 'Classic Dal Lake reflection looking toward Char Chinari. Shoot 30 min before sunrise.' FROM destinations WHERE slug='dal-lake'
UNION ALL SELECT id, 'Nigeen sunset from west bank', ST_GeogFromText('POINT(74.8419 34.1339)'), 'golden-pm',  'W',   true,  false, 'Quieter than Dal. Willows in silhouette against the Pir Panjal.' FROM destinations WHERE slug='dal-lake'
UNION ALL SELECT id, 'Apharwat ridgeline',           ST_GeogFromText('POINT(74.3450 34.0500)'), 'sunrise',    'NW',  true,  false, 'Top of Phase 2 gondola — wide vista of Pakistan-side ranges on clear days.' FROM destinations WHERE slug='gulmarg'
UNION ALL SELECT id, 'Lidder bend at Aru',           ST_GeogFromText('POINT(75.2625 34.0958)'), 'golden-pm',  'W',   false, false, 'River bend where Lidder turns south. Pony stables in foreground.' FROM destinations WHERE slug='pahalgam'
UNION ALL SELECT id, 'Thajiwas glacier mouth',       ST_GeogFromText('POINT(75.2833 34.3167)'), 'golden-pm',  'E',   true,  false, 'Walk 30 min above the glacier viewpoint for the cleanest line.' FROM destinations WHERE slug='sonamarg'
UNION ALL SELECT id, 'Habba Khatoon Peak',           ST_GeogFromText('POINT(74.8419 34.6361)'), 'sunrise',    'N',   true,  false, 'Pyramid peak from Dawar. Dawn light catches the snow first.' FROM destinations WHERE slug='gurez-valley'
UNION ALL SELECT id, 'Tulip terraces',               ST_GeogFromText('POINT(74.8589 34.0908)'), 'sunrise',    'E',   false, false, 'Asia''s largest tulip garden — late March to mid April only.' FROM destinations WHERE slug='tulip-garden';

-- ─── Cultural items (food, festivals, crafts, etiquette) ─────
CREATE TABLE IF NOT EXISTS cultural_items (
  id                     UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  type                   TEXT NOT NULL,
  name                   TEXT NOT NULL,
  name_local             JSONB,
  description            TEXT,
  details                JSONB,
  related_destination_ids UUID[] DEFAULT ARRAY[]::UUID[],
  created_at             TIMESTAMPTZ NOT NULL DEFAULT now()
);

INSERT INTO cultural_items (type, name, name_local, description, details, related_destination_ids) VALUES
  ('dish', 'Wazwan',
   '{"en":"Wazwan","ur":"وازوان","ks":"وازِوان"}'::jsonb,
   'The 36-course Kashmiri feast — Rogan Josh, Gushtaba, Tabak Maaz, Yakhni, all served on a single shared trami plate. A 3-4 hour affair.',
   '{"vegetarian":false,"where_to_try":"Ahdoos · Mughal Darbar · Stream (Srinagar)","price_range":"₹1,500 – 3,000 per person"}'::jsonb,
   ARRAY[]::UUID[]),
  ('dish', 'Kahwa',
   '{"en":"Kahwa","ur":"قہوہ","ks":"قہوہ"}'::jsonb,
   'Green tea with saffron, cardamom, cinnamon and crushed almonds. Drunk through the day, more so in winter.',
   '{"vegetarian":true,"where_to_try":"Any chai stall · Mukhdoom Sahib steps","price_range":"₹20 – 80"}'::jsonb,
   ARRAY[]::UUID[]),
  ('dish', 'Rogan Josh',
   '{"en":"Rogan Josh","ur":"روگن جوش","ks":"روگَن جوش"}'::jsonb,
   'Slow-cooked lamb in a deep-red gravy of Kashmiri chillies, fennel, dry ginger, asafoetida. Not spicy — colour from ratan jot, not heat.',
   '{"vegetarian":false,"where_to_try":"Almost anywhere · Ahdoos first","price_range":"₹400 – 700"}'::jsonb,
   ARRAY[]::UUID[]),
  ('dish', 'Nadru Yakhni',
   '{"en":"Nadru Yakhni","ur":"نَدرو یَخنی","ks":"نَدرو یَخنی"}'::jsonb,
   'Lotus stem in yoghurt curry with fennel and dry ginger. Light, sour, perfect with steamed rice.',
   '{"vegetarian":true,"where_to_try":"Krishna Dhaba · Stream","price_range":"₹250 – 400"}'::jsonb,
   ARRAY[]::UUID[]),
  ('dish', 'Sheermal · Bakerkhani',
   '{"en":"Sheermal · Bakerkhani","ur":"شیرمال · باقرخانی","ks":"شیرمال · باقرخانی"}'::jsonb,
   'Saffron-tinged sweet bread and the flakier, layered Bakerkhani — eaten with butter and kahwa for breakfast.',
   '{"vegetarian":true,"where_to_try":"Old City bakeries · early morning only","price_range":"₹40 – 120"}'::jsonb,
   ARRAY[]::UUID[]),

  ('festival', 'Tulip Festival',         NULL, '1.7 million tulips bloom at the Indira Gandhi garden. Local crafts and Wazwan stalls.',
   '{"month":4,"duration":"2–3 weeks","region":"Srinagar"}'::jsonb, ARRAY[]::UUID[]),
  ('festival', 'Urs-e-Hazratbal',         NULL, 'Annual commemoration at Hazratbal Shrine. Thousands gather. Tourist access very limited that day.',
   '{"month":6,"duration":"1 day","region":"Srinagar"}'::jsonb, ARRAY[]::UUID[]),
  ('festival', 'Eid-ul-Fitr / Eid-ul-Adha', NULL, 'Public holidays. Shops closed. Local hospitality at its peak.',
   '{"month":0,"duration":"Lunar","region":"Whole valley"}'::jsonb, ARRAY[]::UUID[]),
  ('festival', 'Maha Shivratri (Herath)', NULL, 'Important for Kashmiri Pandits. Hazratbal area quiet. Hindus visit Shankaracharya Temple.',
   '{"month":2,"duration":"3 days","region":"Srinagar"}'::jsonb, ARRAY[]::UUID[]),
  ('festival', 'Kheer Bhawani Mela',      NULL, 'Annual gathering at Kheer Bhawani temple, Tula Mula. Pandits offer kheer.',
   '{"month":6,"duration":"1 day","region":"Ganderbal"}'::jsonb, ARRAY[]::UUID[]),

  ('craft', 'Pashmina shawl',     NULL, 'Hand-spun from the undercoat of Changthangi goats. Check for GI tag — most "pashmina" on Lal Chowk is wool blend.',
   '{"origin":"Kanihama","price":"₹4,000 – 80,000"}'::jsonb, ARRAY[]::UUID[]),
  ('craft', 'Papier-mâché',       NULL, 'Layered paper pulp painted with natural pigments. Floral motifs (naqashi). Boxes, ornaments, pen stands.',
   '{"origin":"Old City Srinagar","price":"₹500 – 8,000"}'::jsonb, ARRAY[]::UUID[]),
  ('craft', 'Walnut wood carving', NULL, 'Dense, fine-grained walnut. Furniture, jewellery boxes, lamp bases. Slow to age.',
   '{"origin":"Saderkote","price":"₹1,500 – 50,000"}'::jsonb, ARRAY[]::UUID[]),
  ('craft', 'Hand-knotted carpet', NULL, 'Persian-influenced designs. Knot density (kpsi) determines value. Mostly silk-on-cotton.',
   '{"origin":"Srinagar belt","price":"₹8,000 – 5,00,000"}'::jsonb, ARRAY[]::UUID[]),
  ('craft', 'Sozni embroidery',    NULL, 'Fine needle embroidery on shawls — paisley and chinar motifs. Months of work per piece.',
   '{"origin":"Various","price":"₹2,500 – 25,000"}'::jsonb, ARRAY[]::UUID[]),

  ('etiquette', 'At Hazratbal & Jamia Masjid', NULL,
   'Remove shoes. Women cover head with a scarf (provided if needed). Photography of the relic chamber is forbidden — outside is fine.',
   '{"category":"mosque"}'::jsonb, ARRAY[]::UUID[]),
  ('etiquette', 'Avoid Old City on Friday 12–2 PM', NULL,
   'Jumma prayer crowds. Tourist traffic is unwelcome and you''ll get caught in narrow lanes.',
   '{"category":"mosque"}'::jsonb, ARRAY[]::UUID[]),
  ('etiquette', 'Eat with your right hand', NULL,
   'Wazwan is shared — four people per trami. The eldest takes the first piece. Don''t reach across.',
   '{"category":"wazwan"}'::jsonb, ARRAY[]::UUID[]),
  ('etiquette', 'Don''t leave food on the trami', NULL,
   'Considered wasteful. Pace yourself — there are 36 courses.',
   '{"category":"wazwan"}'::jsonb, ARRAY[]::UUID[]),
  ('etiquette', 'Ask before photographing locals', NULL,
   'Especially women, religious figures, and inside the Old City. A small "Aapki tasveer le sakta hoon?" goes a long way.',
   '{"category":"street"}'::jsonb, ARRAY[]::UUID[]),
  ('etiquette', 'Modest dress is appreciated', NULL,
   'Long sleeves, knees covered at religious sites and in the Old City. A pheran can be bought locally for ₹600 and doubles as warm layer.',
   '{"category":"dress"}'::jsonb, ARRAY[]::UUID[]);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DELETE FROM cultural_items;
DELETE FROM photo_spots;
DROP TABLE IF EXISTS permits;
-- +goose StatementEnd
