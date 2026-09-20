# STRATA — build plan

They were not lost. They were buried, and they have been shouting ever since.

A GPS role-playing game for Android. You fight what the ground remembers, and the gear you take off its corpse belongs to whichever ancient civilization once held the soil under your feet — or to whichever one fills the galleries of the museum across the street.

**Document** — build plan, v0.1 · **Scope** — concept, narrative, systems, data, architecture, cost, risk · **Status** — pre-production, nothing locked

  1. I — Etruria — 900–100 BCE
  2. II — Minoa — 3100–1100 BCE
  3. III — Kemet — 3100–30 BCE
  4. IV — Sumer — 4500–1900 BCE
  5. V — Phoenicia — 1500–146 BCE
  6. VI — Scythia — 900–200 BCE
  7. VII — Hallstatt — 1200–50 BCE
  8. VIII — Meluhha — 3300–1300 BCE
  9. IX — Shang — 1600–1046 BCE
  10. X — Nok — 1500 BCE–500 CE
  11. XI — Olmec — 1600–400 BCE
  12. XII — Chavín — 1200–200 BCE
  13. XIII — Lapita — 1500 BCE–1300 CE
  14. XIV — Hopewell — 1000 BCE–1400 CE
  15. XV — Aksum — 1000 BCE–700 CE

## Contents · What is in here

  1. The game in one page
  2. Story, the fifteen, world coverage
  3. Core loop and session shapes
  4. How the world knows where you are
  5. Light, weather and propagation
  6. Museums as loot beacons
  7. Items, rarity, authenticity
  8. Combat
  9. City services from OSM
  10. Rural, disabled and quiet-city players
  11. Safety and exclusion zones
  12. Android client, 2D
  13. Server architecture
  14. Data pipeline
  15. Anti-cheat
  16. Economy and monetization
  17. Legal and licensing
  18. Cultural guardrails
  19. Roadmap, team, budget
  20. Risk register
  21. Decisions still open

## 01 · The game in one page

Ingress taught players to walk to portals. Pokémon GO taught them to walk to spawns. Strata gives the walking a reason that is already there: the actual history of the ground.

You open the app in Bologna and the monsters that surface are Etruscan — hammer-wielding psychopomps, tomb-wardens, bronze things with mirrors for faces. What they drop is Etruscan: bucchero cups, a fibula, a haruspex's bronze liver. Take the train to Athens and the whole bestiary and the whole loot table change under you. Walk into the Metropolitan Museum in New York and the ground says _nothing_ — but the building says Etruria, Kemet, Shang, and Chavín at once, because that is what is in the cases.

The fiction holds this together. The fifteen did not collapse. They were driven down, over three thousand years, by a society that is still operating. From below, the Interred have built a transmitter out of the only material that reaches the surface everywhere: buried objects. Your phone is a receiver. Every fight is a piece of a message.

### Four commitments that make it a different game

  1. **The map is the content.** No procedural filler dressed as history — civilization zones are drawn from real spheres of influence and real collections, so a trip to Naples, or to the British Museum, genuinely changes what you can obtain.
  2. **Loot is a catalogue.** Every item is a real artefact type with a real card behind it. The collection meta-game is an accidental education, and it is the reason museums will talk to you.
  3. **The city is the UI.** Bakeries heal, banks store, clothes shops let you change loadout, libraries translate. The game reads the city from OpenStreetMap rather than inventing one.
  4. **Single-player first.** No PvP at launch, no raids requiring a dozen strangers. Fights are 45–90 seconds, portrait, one hand, and survive a bad tunnel.

#### On the name

"Strata" is a placeholder chosen because it is short, pronounceable in most markets, and means layers of ground. It is almost certainly taken in software classes — run a trademark and Play Store search before any art is made. Alternates worth clearing: _Chthonia_ , _Underkind_ , _The Interred_ , _Vellum Deep_.

## 02 · Story and the Interred

### The premise

Around 3000 BCE something the buried peoples call _the Concord_ begins organising in the gaps between cities: not a nation, not a religion, a cartel of people who understood before anyone else that written memory is a weapon. The Concord's method is not conquest, it is erasure — silt over a port, a script left untranslated, a library "accidentally" lost, an archive re-attributed to a neighbour. Fifteen peoples resisted long enough to be noticed, and each was, in turn, pushed off the surface entirely. Below, they call themselves _the Interred_ — a name rather than a number, because the count grows every time the Anvil reaches something new.

They did not die underground. They found each other there, across a geology that connects far better than the surface does, and they have spent two millennia building a single instrument — the _Anvil_ — out of everything of theirs still buried above: grave goods, foundation deposits, ballast, scrap. The Anvil cannot send words. It can make a buried object _resonate_ , and it can make the resonance shaped like the thing that killed the civilization it came from.

That is what the monsters are. They are not ghosts and not demons; they are warnings wearing a local costume, because a warning shaped like a Charun gets remembered in Tuscany and a warning shaped like a taotie gets remembered in Anyang. Beating one collapses it back into the object it was made from. That object is your loot, and the fragment of message it carries is your lore card.

#### The turn, for the end of season one

The Concord is not hiding from the app. The Concord is partly running it. The surface telemetry the Interred need — where their artefacts still are, who is near them — is exactly what a Concord front company would want, and the player has been generating it. Season two's question is not "who are the bad guys," it is "whose transmitter is this, and does it matter if the message is true."

### Structure

  * **Codex.** The spine. Each civilization has 40–60 fragment cards; fragments arrive from fights, sites and museums, out of order, and must be _translated_ at libraries and universities to become readable. Translating is a timer, not a puzzle — the puzzle is acquisition.
  * **Season chapters.** Eight to twelve weeks, one civilization in focus worldwide, which lets a Kraków player meaningfully participate in an Olmec chapter through museum beacons and events rather than only through soil.
  * **Personal arc.** One surface NPC — an underemployed conservator who first answered the resonance — and one voice from below per civilization. Keep the cast small and written, not generated.

### The fifteen, and why these fifteen

Chosen against four tests, in this order: **every inhabited region of the planet must fall inside someone's sphere** ; the material culture must be visually unmistakable at 64 pixels; the holdings must be distributed across many countries' museums; and there must be a real historical rupture the fiction can hang on without lying.

Fifteen is the cap, and the constraint that makes the roster work is that a civilization's reach is _core zone plus periphery plus trade corridor_. Phoenicia is a coastline, not a country. Lapita is an ocean. Hopewell is an exchange network that moved Yellowstone obsidian to Ohio. Used properly, fifteen spheres cover the inhabited world with overlap to spare — and overlap is good, because section 4 gives every cell a mixture rather than an owner.

Core roster. Zones are the fiction's simplification of real spheres of influence; the fiction should say so, in the app. Civilization| Soil zone| Museum anchors| Signature drops| Bestiary seed  
---|---|---|---|---  
Etruria| Tuscany, Lazio, Umbria, Po plain, Campania, Corsica| Villa Giulia, Vatican Gregorian, Met, Louvre, British Museum, Volterra| Bucchero ware, bronze mirrors, fibulae, the Piacenza liver| Charun, Vanth, Tuchulcha, tomb-fresco walkers  
Minoa| Crete, Cyclades, Santorini, Aegean coast| Heraklion, Ashmolean, National Archaeological Athens, Met| Labrys, bull rhyton, faience snake figures, Linear A tablets| Labyrinth constructs, bull-daimons, ash-wraiths of Thera  
Kemet| Nile from delta to Nubia, Sinai, Western oases| Grand Egyptian Museum, British Museum, Turin, Neues, Met, Louvre| Was-sceptres, shabti, canopic stoppers, amulets| Shabti swarms, Ammit, Apep echo, quarry-golems  
Sumer| Lower Mesopotamia, Khuzestan, Persian Gulf coast| British Museum, Penn, Louvre, Vorderasiatisches Berlin, Iraq Museum| Cylinder seals, lyre fittings, votive statues, cuneiform tablets| Lamassu sentinels, Humbaba, gallû, canal-things  
Phoenicia| Levant coast plus Carthage, Sicily, Sardinia, Ibiza, Cádiz, Malta| Bardo, Beirut National, Met, Louvre, Cagliari| Murex dye, glass pendant heads, ivory plaques, ship fittings| Glass wraiths, harbour serpents, mask-of-Tanit sentinels  
Scythia| Pontic steppe, Hungary to Altai, north Caucasus| Hermitage, Kyiv Museum of Historical Treasures, British Museum| Akinakes, gorytos, animal-style gold plaques, felt hangings| Golden stags, kurgan riders, Altai griffins  
Hallstatt| Alps, Bohemia, Gaul, Iberia's north, Britain, Ireland| Naturhistorisches Wien, Saint-Germain-en-Laye, National Museum of Ireland, British Museum| Torcs, carnyx, cauldrons, La Tène scabbards| Bog-wights, antlered echo, salt-mine revenants  
Meluhha| Indus basin, Gujarat, Balochistan, Oman trade coast| National Museum New Delhi, Karachi, Mohenjo-daro, Met| Unicorn seals, cubical weights, carnelian beads, bronze figures| Weight-golems, drain-labyrinth crawlers, beast-lord  
Shang| Yellow River, Henan, Shaanxi, Shandong| National Museum of China, Shanghai, NMAA Washington, Academia Sinica, Met| Ding and jue bronzes, jade cong and bi, oracle bones| Taotie devourer, ancestor-bronzes, crack-diviners  
Nok| Jos plateau, Niger–Benue confluence, central Nigeria| National Museum Lagos and Jos, quai Branly, Barakat-free institutional holdings only| Terracotta heads, iron bloom tools, furnace tuyères| Furnace spirits, terracotta sentinels, tsetse-shade  
Olmec| Veracruz, Tabasco, Gulf coast, Chiapas highlands| Xalapa, MNA Mexico City, Dumbarton Oaks, Met| Jade celts, were-jaguar figures, rubber balls, basalt fragments| Were-jaguar, rain-baby, ballcourt revenants  
Lapita| Island Southeast Asia, Melanesia, Micronesia, Polynesia to Rapa Nui and Aotearoa, Madagascar, Taiwan; the whole Pacific and the Indian Ocean rim as corridor| Te Papa, Auckland War Memorial, Bishop Museum, Australian Museum, quai Branly, Field Museum| Dentate-stamped pottery, Tridacna shell adzes, outrigger fittings, barkcloth beaters, star-compass shells| Reef-shades, wayfinder echoes, drowned-island tide-things  
Hopewell| Ohio and Mississippi watersheds, Great Lakes, Gulf coast, Appalachians; corridors to Yellowstone obsidian, Superior copper, Florida shell| Field Museum, Ohio History Center, Cahokia Mounds, Smithsonian NMAI, Peabody Harvard| Mica cut-outs, copper falcon plates, platform pipes, shell gorgets, obsidian bifaces| Earthwork geometers, raptor-spirits, mound-shades  
Aksum| Ethiopian highlands, Eritrea, Red Sea, Yemen and Hadhramaut, Somali and Swahili coast to Mozambique; monsoon corridor to Meluhha| National Museum of Ethiopia, Aksum Archaeological Museum, British Museum, Louvre, Smithsonian NMAfA| Stele fragments, incense burners, Aksumite coinage, Sabaean alabaster heads, ivory| Stele-wardens, incense wraiths, monsoon serpents  
Chavín| Peruvian central coast and highlands, Ancash, Supe valley| Museo Larco, MNAAHP Lima, Chavín site museum, Field Museum| Pututu shell horns, gold crowns, Lanzón motifs, textiles| Lanzón guardian, condor-serpent-feline fusions, gallery echoes  
  
### Planetary coverage audit

Run this table against any map before a region launches. Primary is the civilization whose core or periphery covers the region; secondary is what bleeds in through corridors and gives the local loot table its variety.

Every inhabited region, and who reaches it. Region| Primary| Secondary| Health  
---|---|---|---  
Western & Southern Europe| Hallstatt| Etruria, Phoenicia| Strong  
Italy, Alps, Balkans| Etruria| Hallstatt, Minoa| Strong  
Central & Eastern Europe| Hallstatt| Scythia| Strong  
Scandinavia, Baltic, northern Russia| Hallstatt (amber route)| Scythia forest-steppe| Thin — first fix  
Aegean, Anatolia, Cyprus| Minoa| Phoenicia, Scythia| Strong  
Mediterranean islands and coasts| Phoenicia| Etruria, Minoa| Strong  
Levant, Mesopotamia, Gulf| Sumer| Phoenicia, Meluhha| Strong  
Iran, Caucasus, Central Asia| Scythia| Sumer, Meluhha| Good  
Egypt and the Nile to Nubia| Kemet| Aksum, Phoenicia| Strong  
Maghreb and Sahara| Phoenicia (Carthage)| Kemet, Nok (trans-Saharan)| Good  
West and Central Africa| Nok| Phoenicia, Aksum| Good  
Southern Africa| Nok (Bantu expansion corridor)| Aksum (Swahili coast)| Adequate  
Horn of Africa, Red Sea, Arabia| Aksum| Kemet, Sumer| Strong  
South Asia| Meluhha| Aksum (monsoon), Scythia (north)| Strong  
Siberia, Mongolia, the steppe| Scythia| Shang| Good  
China, Korea, Japan| Shang| Scythia, Lapita (southern coast)| Good; Jōmon is the season-two fix for Japan  
Mainland Southeast Asia| Lapita| Shang, Meluhha| Adequate  
Island SE Asia, Oceania, Aotearoa, Madagascar| Lapita| Meluhha| Strong  
Australia| Lapita, northern coasts| Meluhha, Shang by drift| No core by design; drift-filled, see below  
Eastern and central North America| Hopewell| Olmec (Gulf)| Strong  
Western North America| Hopewell (obsidian corridor)| Olmec| Adequate; Ancestral Puebloan is the fix  
Mesoamerica and the Caribbean| Olmec| Chavín, Hopewell| Strong  
Andes and Pacific South America| Chavín| Olmec| Strong  
Amazon and Atlantic South America| Chavín (thin)| Olmec| Thin — Marajoara is the fix  
Arctic, Antarctic, mid-ocean| Drift only| Whatever is nearest, blended| Deep drift: the rarest hybrids in the game  
  
#### Nowhere is empty: the drift

No cell on the planet has a weight of zero. Where no core reaches, the surrounding spheres **bleed outward** — a distance-decayed field over land, coast and ocean, described in section 4 — so that Nuuk, Perth, Ulaanbaatar and a container ship in the mid-Atlantic all resolve to a real mixture of real civilizations, just a faint and blended one.

The fiction earns it twice over. The Anvil's signal travels through rock and water and refracts on the way; far from its source it arrives smeared together with whatever else is passing through. And the surface truth is the same story: objects have always moved. Ballast, trade, migration, tribute, loot, colonial collecting. A Benin bronze in Berlin and a Phoenician bead in a Baltic grave are both drift, and neither is a fantasy.

What drift produces is _hybrid_ gear — a Shang silhouette carrying Lapita stamping, an Etruscan form in Hopewell mica. Every region has its own stable blend, which means the emptiest places on the map become the _only_ source of their particular combination. A player in Alice Springs or Iqaluit is not playing a degraded version of the game; they are sitting on something nobody in Rome can farm.

Where drift is at its faintest — mid-ocean, the ice, deep desert — one more thing unlocks. Banked resonance can be spent as an _uplink_ : transmitting downward instead of receiving, the only way to send the Interred information rather than take it, and the only place certain codex answers come back. It is the content that belongs to ships, remote postings and long flights.

#### Australia, and the limits of the premise

Aboriginal and Torres Strait Islander cultures are the oldest continuous living cultures on earth. A fiction in which a people was driven underground and vanished is the wrong shape for them, and no amount of respectful art direction fixes a premise mismatch. The drift solves the map problem without solving the wrong problem: the continent fills with a Lapita-Meluhha-Shang blend, plus museum beacons, and stays full of content without anyone inventing a vanished Australian civilization. Three options, in order of preference: ship the drift fill and leave the core slot deliberately empty; commission a bespoke, non-disappearance treatment with Indigenous-led organisations and pay properly for it; or represent Australia only through institutional holdings. Do not quietly invent a fifteenth-and-a-half civilization to fill the map.

The same test applies everywhere the roster touches living descendant communities — Hopewell especially, where descendant nations are federally recognised and actively engaged in repatriation. Consult before you write, not after.

#### Held for later seasons, in priority order

Nordic Bronze Age (fixes Scandinavia), Jōmon (fixes Japan), Ancestral Puebloan (fixes the American southwest), Marajoara (fixes Amazonia), then Hatti, Sanxingdui, Cucuteni–Trypillia, Great Zimbabwe, Tartessos, Rapa Nui. Sanxingdui is the season-three reveal: a genuinely unexplained bronze culture with masks that look designed for this game.

## 03 · Core loop and session shapes

Design for three session types with completely different postures, because location games fail when they assume everyone is on a dedicated walk.

Session| Length| Posture| What it must deliver  
---|---|---|---  
Check-in| 60–120 s| Standing at a bus stop, one hand| Claim nearby resonance, one fight, bank a drop, out  
Commute| 10–25 min| Walking or on transit, screen mostly dark| Passive collection along the route, a queue of fights to resolve later, step-driven progress  
Expedition| 45–120 min| Deliberate trip: old town, site, museum| Rare beacons, chapter content, the reason to travel  
  
### The loop

  1. **Sense.** The map shows resonance within ~250 m, weighted by the civilization mix of your cell.
  2. **Engage.** Walk into range (40 m), fight a 45–90 s battle.
  3. **Collapse.** The monster reduces to an object: an item plus, sometimes, a codex fragment.
  4. **Sustain.** Health is spent by fighting, restored at food POIs. This is the pacing throttle, and it is why the city layout matters.
  5. **Consolidate.** Bank surplus at a vault, re-equip at a clothes shop, reforge at a hardware store or smith, translate at a library.
  6. **Read.** A fragment becomes readable; the chapter advances; a new zone or beacon opens.

**Passive collection matters more than anything else on this list.** Most users will not open the app during their commute. A foreground-service step and location tracker that quietly accrues resonance along the route, then presents "you walked through 2.1 km of Hallstatt ground, 4 things followed you home" on next open, is the retention mechanic. Budget it as a first-class feature, not a nice-to-have, and budget engineering time for Android background-execution restrictions per OEM.

## 04 · How the world knows where you are

### Civilization weight, not civilization ownership

Every location resolves to a **weight vector** over the fifteen, not to a single owner. Rome is Etruria 0.55 / Phoenicia 0.10 / Kemet 0.05 with a long tail; the Met's footprint is a near-even split across nine. This single decision avoids most of the design problems: borders stop being arbitrary lines, trade civilizations like Phoenicia can bleed along entire coasts, and museums compose naturally with soil.

#### Layers, summed in this order

  1. **Core zone polygons** — hand-drawn GeoJSON per civilization, core and periphery rings. Hand-drawn is correct here: archaeological reality is fuzzy, and a cartographer plus a consulting archaeologist will produce something more defensible and more playable than any automated extraction.
  2. **Trade corridors** — line geometries buffered 10–40 km: the Phoenician littoral, the steppe belt, the amber and tin routes, the Nile, the Gulf run to Meluhha.
  3. **Site points** — `historic=archaeological_site` and curated major sites, radial falloff 0.5–3 km, high intensity.
  4. **Museum beacons** — see section 6.
  5. **Drift field** — a computed diffusion from every core outward, filling everything the first four layers miss. This is what guarantees the sum of weights is never zero.

### The drift field

Solve it once, offline, as a cost-distance problem on the global cell graph. For each civilization, run a multi-source Dijkstra from its core cells with anisotropic movement costs:

  * **Land** cheap; mountain, ice cap and hyper-arid desert expensive; navigable river valleys and old route corridors cheapest of all.
  * **Coastline** cheap for the maritime fifteen — Phoenicia, Lapita, Minoa, Aksum — and expensive for the landlocked ones. Scythia does not drift across the Pacific; Lapita does almost nothing else.
  * **Open ocean** expensive for everyone, and multiplied by each civilization's own reach constant, so the Pacific ends up Lapita's with a thin Chavín and Shang presence, which is both good design and defensible history.

Convert cost to weight with an exponential decay per civilization, then normalise:
    
    
    w_i(cell) = exp( -cost_i(cell) / lambda_i )
    weights   = normalise( top4(w) )          // keep 4, fold the tail into a residual
    purity    = w_max / sum(w)                // 1.0 at a core, ~0.3 mid-ocean
    

The decay constant `lambda_i` is not fixed — section 5 makes it move with time of day and weather, which is what gives conditions their teeth.

**Purity is the second number every cell carries** , and it drives everything downstream: high purity yields clean single-civilization items, low purity yields hybrids, and the lowest band unlocks the uplink. It also gives balance one dial to pull instead of fifteen.

Cost: fifteen Dijkstra runs over a land-and-shelf graph, plus a coarse ocean graph. Run it at H3 resolution 6, interpolate up to resolution 8, and it is a nightly batch job measured in tens of minutes, not hours. Store per cell: four civilization ids, four quantised weights, one purity byte — nine bytes, which keeps the whole planet comfortably in memory.

#### Precomputation

Do not run point-in-polygon per request. Precompute the weight vector for every **H3 resolution 8 cell** (~0.74 km², ~0.46 km edge) on land, store as a compact array, and serve from an in-memory or Redis lookup. H3 r8 over inhabited land is on the order of tens of millions of cells — a few GB, trivially shardable, and recomputed offline as a batch job whenever zones or museum data change. Spawn resolution is r9 or r10 for placement precision; weight resolution stays r8.

#### Deterministic spawns instead of a spawn table

Never store individual monsters. Given `(h3_cell, epoch_15min, server_seed)`, hash to a PRNG and derive the spawn set from the cell's weight vector and its POI density. Any server can answer any query about any cell without shared state, spawns are identical for all players in a cell (which makes co-located play feel real), and the cost is a hash. Persist only what players actually interact with: claims, kills, drops.
    
    
    spawns(cell, epoch) =
      seed  = siphash(server_secret, cell_id, epoch)
      n     = clamp(base_density * poi_density_factor(cell), 0, 12)
      for i in 0..n:
         civ   = weighted_pick(weights[cell], seed, i)
         tier  = tier_curve(civ, player_chapter, seed, i)
         point = jitter_within(cell, seed, i)   // snapped off roads/water/private
    

## 05 · Light, weather and propagation

The world should not be a flat texture with a night filter on top. Time of day, weather, tide and terrain feed one scalar and one condition vector, and every spawn rule in the game reads them.

### The hook, which is real physics

AM radio travels far further at night: the ionosphere's D layer thins after dark and signals that die 100 km out at noon skip across continents at 2 a.m. Wet ground conducts better than dry. And during a total solar eclipse the ionosphere briefly behaves as though night had fallen — a measurable effect that amateur radio operators run coordinated experiments on during every eclipse.

The Anvil is a transmitter. So night, rain and eclipse do exactly one thing to it: **they increase propagation**. That single idea makes weather matter to loot rather than to skins, and it comes out of the drift field you already built.

### Propagation, and the trick that makes it cheap

Propagation is one scalar, 0 to 1, per cell per epoch. It stretches the decay constant in section 4's drift solve:
    
    
    lambda_i(cell, t) = lambda_i_base * (1 + k_i * propagation(cell, t))
    w_i               = exp( -cost_i(cell) / lambda_i(cell, t) )
    

**The costs are static; only lambda moves.** The expensive part — fifteen cost-distance solves over the planet — stays a nightly batch job. Store `cost_i` per cell for the nearest few civilizations plus a residual, and recomputing weights under new conditions is a handful of exponentials at request time. Conditions become free.

Condition| Propagation| What the player sees  
---|---|---  
Midday, dry, clear| Low| Short reach. High purity: local, clean, single-civilization items. The best conditions for Grounded and Sited hunting.  
Dusk and dawn| Medium, rising| The shift window; both faces briefly available.  
Night| High| Distant civilizations reach you. Hybrids and Confluence become likely. The far side of the world bleeds in.  
Rain, wet ground| Raised| Local signal strengthens; more spawns, better tiers, worse purity.  
Thunderstorm| Spiking, unstable| Violent spawns, high yield, status effects on the player.  
Snow and hard frost| Suppressed| Quiet map, rare specific spawns, a reason to go out anyway.  
Fog| Distorted| Sense radius collapses to 80 m; things arrive without warning.  
Total eclipse| 1.0, for minutes| See below.  
  
### The condition vector

Field| Source| Refresh| Cost  
---|---|---|---  
Solar altitude, twilight phase, day length| Computed (NREL solar position algorithm)| Continuous| Free, exact, works offline  
Moon phase, altitude, illumination| Computed| Continuous| Free  
Season, solstice and equinox proximity| Computed| Daily| Free  
Tide phase and range| Harmonic constants, coastal cells only| Computed forward| Free; check the licence on your constants set  
Temperature, precipitation, wind, cloud, visibility, thunder| Open-Meteo or MET Norway| 20 min TTL| Free tier with attribution, if you cache properly  
Precipitation and temperature _anomaly_ vs local normals| Climate normals baked into the cell store| Static| Free  
Geomagnetic Kp index (aurora)| NOAA SWPC| 15 min, global| Public domain, one call  
Water proximity, forest, elevation band, coastline, urban density| OSM plus Copernicus DEM, baked| Nightly| Already in the pipeline  
Active event ids| Event service| Per epoch| —  
  
#### Weather without a bill or a bottleneck

Never one provider call per player. Key the weather cache on **H3 resolution 5** (~250 km², fine enough for a mountain-versus-valley difference), fetch only for cells that have had an active player in the last fifteen minutes, hold a 20-minute TTL with stale-while-revalidate, and fall back to fair weather rather than blocking a spawn query when the provider is slow or down. At 100k daily actives that is a few thousand hot cells rather than millions of calls, which sits inside the free tiers with attribution. Budget for a paid provider anyway from soft launch; it is a small line and an outage should not degrade the game.

#### Server authority

Conditions are computed server-side from the cell's own coordinates and UTC, pinned at the start of each 15-minute epoch and shipped inside the cell payload. The client recomputes astronomy locally for presentation so the sky looks right offline, but it can never assert that it is night or storming where it stands. A device clock is not evidence.

### Spawn rules as data, not code

Every monster, item bias and ambient effect declares predicates over the condition vector. Rules live in a versioned table, hot-reload without an app release, and are what the design team actually ships week to week.
    
    
    {
      "id": "minoa.tide_daimon",
      "civ": "minoa",
      "when": { "all": [
          { "is_night": true },
          { "water_proximity_m": { "lt": 120 } },
          { "moon_illumination": { "gte": 0.8 } },
          { "any": [ { "tide_phase": "flood" },
                     { "precip_anomaly_pct": { "gte": 70 } } ] }
      ]},
      "weight": 3.0,
      "weight_mods": [ { "if": { "propagation": { "gte": 0.7 } }, "mul": 1.8 } ],
      "cap_per_cell": 1,
      "loot_bias": { "archetype": "off.vessel", "tier_shift": 1 }
    }

Predicates compile to a bitmask test over a packed condition struct, so evaluating several hundred rules for a cell is microseconds; memoise the eligible rule set per condition digest, since every cell in a region shares one. Then fold that digest into the existing spawn hash — `spawns(cell, epoch, digest)` — so the world changes when conditions change but never flickers inside an epoch.

### Fairness, which is where this usually goes wrong

  * **Use anomalies, not absolutes.** "Rain" as a threshold in millimetres means Bergen sees nothing else and Lima never sees it once. Trigger on percentile against local climate normals, and a wet week in the Atacama counts as a storm.
  * **Every condition-locked collectible needs a second path** — seasonal rotation, uplink, or accession — or you have shipped items that are unobtainable by latitude. Someone in Singapore should not be locked out of the frost line forever.
  * **Night is not premium content.** Most people play in the evening. Day and night are two equal faces of the same game, not a base game and a bonus.
  * **Night interacts with safety.** Section 11 damps spawns in unlit areas after midnight; night's propagation bonus should therefore concentrate its rewards where people already are — lit streets, home, transit — rather than pulling anyone into a dark industrial estate.

### The event framework

One object type, from a season-long chapter down to a four-minute eclipse, with no special cases:

  * **Geometry** — polygon, corridor, cell set, moving swath, or global.
  * **Window** — absolute times, or _per-cell_ times to the second for events whose geometry moves.
  * **Trigger** — scheduled; astronomical (computed, so eclipses and meteor peaks are known years ahead); conditional ("first thunderstorm of the season in this cell", "Kp ≥ 6 and clear skies"); live-ops manual; or player-caused.
  * **Effects** — propagation override, spawn table injection, loot modifiers, a unique boss, a currency, medal hooks.
  * **Stacking** — integer priority, at most two visible at once, deterministic tiebreak.
  * **Notification policy** — scheduled events announce weeks out; short-notice ones like aurora need a separate surprise path with per-player quiet hours respected.

### Eclipses: the case that proves the framework

  * **Geometry.** Derive the umbral path from Besselian elements — NASA publishes them, public domain, decades ahead — into per-cell local contact times at second precision, plus an obscuration scalar for the whole partial hemisphere.
  * **Behaviour.** Inside totality, propagation snaps to 1.0 for the local duration. For those two to four minutes the entire planet is adjacent: a Lapita hybrid can drop in Ohio, and nothing else in the game does that.
  * **Nobody is excluded.** Propagation scales with obscuration, so a continent participates at lower intensity, and a player at 85% coverage 400 km off the centreline still has a remarkable afternoon.
  * **Offline-first is mandatory, not a nicety.** A totality path pulls hundreds of thousands of people into rural cells and the mobile network collapses — this is reliably observed at every eclipse. The client pre-caches a signed event bundle days ahead containing the rules, the art, the audio and its own local contact times, validates and plays entirely offline, queues claims, and the server adjudicates on reconnect against a path it already knows. If it does not work with no signal, it does not work.
  * **Consistency with the medal rules in section 12.** Timed-and-placed incentives were ruled out because manufactured urgency makes people hurry. A natural event is different, but only under conditions: announced weeks ahead, participating from any point in the band including your own garden, no requirement to reach a particular spot by a deadline, and **no medal, season pass or codex keystone ever gated on catching a four-minute window**. Something rare and lovely. Never something anyone would drive dangerously for.
  * **Say the obvious thing.** The app tells people not to look at the sun without certified filters, every time. As it happens, the fight is on the phone, which is a legitimately good place for their eyes to be.

#### Build cost

Astronomy library about a week. Condition service and cache, two weeks. Rule language and evaluator, two to three. Event framework, three. Eclipse pipeline, two, and it can be done any time before the next one. Sequence it as day/night plus the condition vector in phase 1, weather and the rule language in phase 2, the event framework and the first eclipse in phase 3 — by which point live ops is the team's main job anyway.

## 06 · Museums as loot beacons

The feature nobody else has: press, partnerships, and a defensible reason to leave the house. It is also the one that needs the most unglamorous data work.

#### Scoped as a feature, not the thesis

Museums are one of four ways ground gets its weight — soil, corridors, sites, beacons — and the game has to be complete for a player who never enters one. Four consequences, all of them budget: curate **150 institutions** by hand, not 300, and let Wikidata carry the long tail; defer gallery-level beacons until a partner asks for them; keep **no institution on the critical path to launch** , so a slow legal department cannot move your date; and keep remote accession generous so the mechanic reads as a bonus rather than a gate.

### Getting museum-to-civilization mappings

  1. **Wikidata SPARQL, automated.** Query objects with a collection statement (`P195`) and a culture or material-culture statement (`P2596` / `P361`), group by institution and culture, and you get a first-pass holdings count per museum per civilization for several thousand institutions. Noisy but excellent for breadth.
  2. **Open museum APIs, for depth.** The Met's Open Access dataset carries a culture field per object; the Rijksmuseum, Smithsonian, Cleveland, Harvard Art Museums, Europeana and the Victoria and Albert all publish usable APIs or CSV dumps. These give you real object names and, often, CC0 imagery — which is a legitimate path to authentic item art for a fraction of an art budget.
  3. **Manual curation, for the top 150.** One researcher, three to four weeks, produces a hand-checked table of the world's significant collections per civilization with gallery-level placement where possible. Everything downstream of that table gets better.
  4. **Player-proposed additions** with moderation, from season two.

### Beacon mechanics

  * A museum footprint (OSM building polygon, buffered 60 m) carries a weight vector derived from its holdings, capped so that one mega-museum does not trivialise the whole game.
  * Entering grants an _accession_ : a limited number of high-tier pulls from that museum's civilizations, per museum, per 20–24 h, gated on a genuine dwell time of several minutes inside the polygon.
  * Gallery-level beacons for partnered institutions: separate rooms grant separate accessions, which is a real reason to walk the whole museum, and is exactly the behaviour museums want to buy.
  * Items pulled at a museum are marked with the institution, permanently, on the item card. Collection-driven players will chase provenance sets across cities. This is the long-tail retention hook.
  * **Do not require the paid ticket.** Grant a reduced accession from the forecourt so that the mechanic is not a paywall; grant the full one inside.

#### Rules that keep museums on your side

  * Nothing in the game should reward moving fast, filming, or touching anything. Fights inside museum polygons switch to a quiet mode: no sound by default, longer timers, an explicit "you are in a gallery" framing.
  * Any institution can opt out or restrict to forecourt with an email. Build the switch before launch, not after the first complaint.
  * Never place a beacon on a memorial museum, a site of atrocity, or a house of remembrance. Maintain a hard blocklist, seeded manually, reviewed by a human. This is the single fastest way to destroy the project's reputation.
  * Objects with contested acquisition histories — Nok terracottas and Benin material especially — should carry the provenance question _in the lore card text_ rather than being quietly excluded. The fiction is about erasure; a game about erasure that hides looting is embarrassing.

## 07 · Items, rarity, authenticity

### Item model

An item is `civilization × archetype × tier × affixes × provenance`. Archetypes are shared across civilizations so balance is tractable; flavour, art and stat bias come from the civilization.

Slot| Archetype| Etruria example| Shang example| Scythia example  
---|---|---|---|---  
Main| Edge / Haft / Focus| Kopis, ritual hammer| Bronze _yue_ axe| Akinakes  
Off| Guard / Vessel| Bronze mirror| Jade _bi_ disc| Gorytos  
Body| Wrap / Plate| Linen thorax| Lacquered leather| Scale coat  
Head| Crown / Mask| Negau-type helm| Taotie mask| Bashlyk  
Charm ×2| Amulet / Seal| Bulla| Oracle bone shard| Stag plaque  
Relic| Set keystone| Haruspex liver| Ding cauldron| Pectoral  
  
#### Rarity, and the authenticity axis

Two independent axes, which is unusual and worth keeping:

  * **Power tier** — Common, Burnished, Votive, Funerary, Anvil-touched. Drives stats.
  * **Authenticity** , read off the cell's purity — _Drift_ (hybrid, from anywhere the cores do not reach), _Confluence_ (three or more spheres at near-equal weight: rare, and geographically specific), _Grounded_ (the civilization's own soil), _Accessioned_ (a museum that holds it), _Sited_ (a real archaeological site inside the zone). Drives codex value, cosmetic finish and trade value — _not_ raw power.

Separating them is what stops the game from telling a player in Manitoba that they cannot compete. They can reach maximum power locally; what they cannot get without travelling is the Sited Etruscan mirror with the Volterra stamp on it. Prestige is geographic, power is not.
    
    
    {
      "id": "etr.mirror.votive.1a3f",
      "civ": "etruria",
      "archetype": "guard.mirror",
      "tier": "votive",
      "authenticity": "accessioned",
      "provenance": { "kind": "museum", "osm_id": "w/12345678",
                      "name": "The Metropolitan Museum of Art",
                      "acquired_at": "2026-09-16T14:02:11Z" },
      "affixes": [ {"k":"ward","v":18}, {"k":"resonance_on_collapse","v":6} ],
      "codex_ref": "etr.f027"
    }

#### Hybrids, and why they are cheap to make

A drift item is an ordered pair: dominant civilization supplies the archetype and silhouette, secondary supplies pattern, palette and one prop. _Tide-worn ding, Shang carried on Lapita water._ _Mica kopis, Etruria arriving in Hopewell._ Because the art is 2D and layered, a hybrid is a composite of assets you already drew — silhouette layer, pattern fill, palette map, prop overlay — not a new illustration. Fifteen civilizations give 210 ordered pairs; across seven archetypes that is fifteen hundred recognisably distinct items out of roughly a hundred hand-drawn pieces. This is the single biggest argument for the 2D decision in section 12.

Constrain it so it does not become slop: only pairs that are geographically plausible under the drift field ever occur, each region's blend is stable rather than rolled per drop, and hybrid names come from a written table per pair, never from a generator.

#### Hybrids must never pretend to be real

Section 18 requires that a player can always tell invention from fact. A hybrid is invention end to end, so its card carries no non-fiction panel at all — just the Anvil-fiction, in the fiction typeface, with an explicit line saying this object was made by the transmission and never existed above ground. The two real objects it was composed from are linked, with their real cards intact. Handled this way the hybrid actually teaches something true about diffusion; handled carelessly it is a machine for inventing fake archaeology, and museums will notice which one you built.

#### Lore cards

Every archetype carries 80–150 words written by someone who knows the material: what the object actually was, who used it, what is genuinely unknown about it, and then one line of Anvil-fiction. Keep the real and the invented visually separated on the card — a different type style for the fiction. Players will screenshot these; that is free marketing and it must be accurate.

## 08 · Combat

Constraints first: portrait, one thumb, 45–90 seconds, playable while standing, survives a 4-second network gap, and never asks the player to stare at the screen while moving.

### Shape

Real-time with a cadence, not a twitch game. Each combatant has a rhythm bar; on your beat you choose one of four actions. Reading the monster's tells and spending your beats correctly is the skill. It is closer to a fighting-game read than to an RPG menu.

  * **Strike** — damage, weapon-typed.
  * **Ward** — reduce and counter the telegraphed attack; timing window.
  * **Invoke** — spend Resonance on a civilization ability from your set bonuses.
  * **Read** — reveal the monster's next two beats and build Resonance. The skill-expression action.

#### Stats

Four only: **Might** (damage), **Ward** (mitigation), **Resonance** (ability economy and codex yield), **Fortune** (drop tier and affix quality). Material triangle on top: _Bronze_ beats _Stone_ , Stone beats _Gold_ , Gold beats Bronze, at ±25%. Each civilization biases toward a material but is not locked to one, which makes mixed loadouts viable and gives the travelling player something to optimise.

#### Authority

Server-authoritative and deterministic. The client runs the same simulation for responsiveness and sends an input stream with beat indices; the server replays it and returns the canonical result plus the drop. Mismatches beyond tolerance are logged and, if repeated, the account is flagged. Never let the client tell the server what it looted — that is the single most-exploited surface in every location game shipped so far.

#### Accessibility, non-optional

An auto-resolve mode that plays the fight with your loadout at a modest yield penalty. Blind and low-vision players, players with motor impairments, and people on a crowded train all need it, and it costs you one function since the simulation is already deterministic and server-side.

## 09 · City services from OpenStreetMap

Tag-to-function mapping. Each needs a fallback for places where the tag is rare, and each needs a cooldown so that living above a supermarket is not a strategy.

Range is the interaction radius from the POI's point or polygon edge. Function| OSM selectors| Effect| Range| Cooldown  
---|---|---|---|---  
Hearth (heal)| `shop=bakery|pastry|convenience|supermarket|greengrocer|butcher`, `amenity=cafe|fast_food`| Restore health; bakeries restore most, per capita of charm| 35 m| 30 min per POI, 6 h soft daily cap  
Vault (storage)| `amenity=bank|bureau_de_change|post_office`, `amenity=atm` (reduced)| Deposit and withdraw from stash; inventory is deliberately small| 30 m| None; withdrawals rate-limited  
Wardrobe (loadout)| `shop=clothes|boutique|shoes|jewelry|tailor|department_store|second_hand`| Change equipped set, apply cosmetic finishes| 30 m| None  
Forge (upgrade)| `shop=hardware|doityourself|locksmith`, `craft=blacksmith|metal_construction|jeweller`| Reforge affixes, raise tier with materials| 30 m| Material-gated  
Apothecary (cure)| `amenity=pharmacy`, `shop=chemist|herbalist`| Clear status effects: corrosion, silt-blind, oath-bound| 30 m| 2 h  
Scriptorium (translate)| `amenity=library|university|college|archive`, `shop=books`, `tourism=information`| Start and collect codex translations; parallel slots| 50 m| Slot-gated  
Beacon (loot)| `tourism=museum|gallery`, `historic=archaeological_site|ruins|monument|memorial*`, `man_made=obelisk`| Accession pulls; see section 6| Polygon + 60 m| 20–24 h  
Caravan (travel)| `railway=station`, `aeroway=terminal`, `amenity=ferry_terminal|bus_station`| Register a journey; arrival grants a first-contact bonus in the new zone| polygon| Per journey  
Spring (energy)| `amenity=drinking_water|fountain`, `natural=spring`, `man_made=water_well`| Restore Resonance out of combat| 25 m| 45 min  
Wild density| `leisure=park|nature_reserve`, `natural=wood|beach`, `landuse=forest`| Raises spawn count, lowers tier: grinding grounds| polygon| —  
  
* `historic=memorial` only after human review. Most memorials must be excluded. See section 11.

#### Practical notes

  * **Chains and franchises.** Use the `brand:wikidata` tag to normalise chains, then cap how much a single chain contributes in one city, or the game becomes "walk to the nearest Lidl" in half of Europe.
  * **Opening hours.** OSM's `opening_hours` is machine-parseable and about 60–70% present in well-mapped areas. Respect it where present — a closed bakery healing you is a small immersion break, but a closed bakery _not_ healing you when the tag is simply missing is a bug report. Default to open when the tag is absent.
  * **Give back.** A "this shop is gone / this is mistagged" report flow that feeds a moderated queue, with a proper OSM changeset account and a published editing policy. The OSM community will notice either way; being a good contributor is cheaper than being a nuisance.

## 10 · Rural, disabled and quiet-city players

A game keyed to POI density punishes the countryside, punishes anyone who cannot walk far, and punishes cities where OSM coverage is thin. Fix it in the design, not with an apology.

  * **Density normalisation.** Compute POI density per H3 r8 cell and scale yields inversely within bounds. A monster in a village is rarer and worth several times a city monster. Dense-urban players get volume, rural players get value; both reach the same weekly ceiling.
  * **Synthesised nodes.** Where a cell has no qualifying service POI within 800 m, generate a persistent, server-authored _cairn_ at a suitable public location — a road junction, a church forecourt, a trail head — that provides the missing function. Never place on private residential land.
  * **Seated play.** Every daily objective must be completable within a 200 m radius. Distance-based objectives are always optional and always have an alternative.
  * **The Anvil at home.** A home anchor you can set once per week that slowly draws resonance toward it, so that a housebound player still has an inbound stream.
  * **Remote accession.** A small number of museum pulls per season usable from anywhere, framed as an archive request. This is the housebound player's access to the travel content, and it should be generous.

## 11 · Safety and exclusion zones

**Treat this as a launch blocker, not a policy page.** The location-game genre's worst moments — players at Auschwitz, at Arlington, in hospital wards, trespassing on private property, hit by cars — all came from the same omission: content placed by an algorithm without a human veto.

#### Hard exclusions, no gameplay of any kind

`amenity=school|kindergarten|childcare|hospital|clinic|police|fire_station|prison|place_of_worship*`, `landuse=military|residential(building=house|apartments)`, `historic=memorial` where the memorial commemorates death or atrocity, all cemeteries and graveyards, level crossings, motorways and their verges, active railway, airport airside, and anything inside a curated blocklist of sites of conscience.

* Places of worship are opt-in per site, via the same institutional contact flow as museums. Many will happily host; none should be enrolled without asking.

#### Motion and attention

  * Speed gate: above roughly 15 km/h sustained, interaction is disabled and the app shows the passive collection view only. Detect with activity recognition, not raw GPS alone, so transit passengers can be offered a "I am a passenger" confirmation that unlocks a reduced mode.
  * No timed content that rewards hurrying to a location. Windows are hours, never minutes.
  * Night dampening after local midnight: reduced spawn density in unlit residential areas, and no beacon rewards that would put a player in an empty industrial estate at 2 a.m.
  * An in-app report for a dangerous or inappropriate node, with a human queue and a 48-hour SLA.

## 12 · Android client, 2D and native

Decided: Kotlin and Jetpack Compose, MapLibre Native for the world map, 2D throughout, no Unity, no AR at launch.

### What 2D buys you

  * **A 45 MB app instead of 180 MB.** On the mid-range Android hardware this game actually lives on, install size is a conversion metric.
  * **Battery you control.** Passive collection along a commute is the retention mechanic (section 3), and it needs a plain foreground service and the Fused Location Provider — not an engine's main loop fighting Doze.
  * **Fifteen civilizations become affordable.** Illustration scales by drawing; 3D scales by modelling, rigging and texturing. At this roster size, 2D is not a compromise, it is the only way the content budget closes.
  * **Art direction that can be strange.** Flat, graphic, drawn from the source material — Etruscan tomb-fresco line, Shang bronze relief, Lapita dentate stamping, Hopewell mica silhouettes. Fifteen distinct visual languages, each unmistakable at thumbnail size. Mid-poly 3D would flatten all fifteen into the same plastic.

### Stack

  * **UI, menus, inventory, codex** — Compose. Fast to iterate, accessible by default, themable for the fifteen.
  * **World map** — MapLibre Native with self-hosted vector tiles as PMTiles on a CDN, a hand-made dark style, and an offline cache. Spawn and POI markers are a Compose overlay above the map surface, not map symbols, so animation stays under your control.
  * **Combat** — its own view, one of two renderers. Prototype both in week one of phase 0 and keep the winner: 
    * _Rive_ — vector animation with state machines, tiny files, a solid Android runtime, and artists can author the monster's whole tell-and-react behaviour without an engineer. Best fit for the read-the-tells combat in section 8.
    * _Compose Canvas with sprite atlases_ — no dependency, total control, more engineering per monster. The safe fallback.
Spine is the third option if you hire animators who already use it; it is a per-seat licence and a heavier runtime.
  * **Location** — Fused Location Provider, balanced power in passive mode, high accuracy only in a fight or near a beacon; `ACTIVITY_RECOGNITION` for the speed gate.
  * **Local state** — Room mirroring server state, with an operation queue the server replays and adjudicates on reconnect.
  * **Transport** — protobuf over HTTP/2; one WebSocket for live cell state while foregrounded.
  * **Platform** — minSdk 26, Play Games Services v2, Play Integrity, Play Billing, FCM.

### Play Games Services: identity, achievements, leaderboards

Play Games v2 is the login, and that is a good call for a game people will open at a bus stop: automatic sign-in, no email wall before the first fight, free cross-device continuity, and an install base that is already signed in. Three parts, each with a trap.

#### Identity — do this so it does not trap you later

  * **Server-side auth, always.** The client calls `requestServerSideAccess`, gets a one-time auth code, and posts it to your server; the server exchanges it and resolves the player id against Google itself. Never accept a player id from the client — it is a plain string and treating it as identity is the classic account-takeover bug in this genre.
  * **Your own account id from day one** , with an identity table of `(provider, provider_id, account_id)` and Play Games as merely the first row. It costs a day now. When the iOS client arrives, or when someone wants to link an email, the alternative is migrating every player record and every provenance stamp.
  * **Recall API** for the player who changes Google account or loses a device. It exists for exactly this and it is much better than the support queue you get without it.
  * **Server is authoritative; no Snapshots.** Do not use Play Games saved games as a source of truth — inventory, codex and provenance live in Postgres. Play Games is a display and identity surface, not a database.
  * **Opt out of Google Play Games on PC distribution.** Enabling Play Games normally makes an app a candidate. An emulator with a settable location is the cleanest spoofing environment that exists, and you would be shipping it yourself.

#### Medals and achievements — around fifty at launch

A **cumulative tiered medal is not a leaderboard** , and the distinction matters more than the earlier draft allowed. A ranked distance board is zero-sum: your kilometre devalues mine, the top belongs to whoever has the most free time or the best spoofer, and everyone else is scenery. A medal at 25 / 100 / 500 / 2,000 / 10,000 km is a personal ratchet — nobody loses when you gain, the ceiling is years away, and Ingress has a decade of evidence that the health line works. Trekker got people outdoors. Build it.

Five tiers per medal — _Sherd, Bronze, Silver, Gilt, Onyx_ — across three families:

  * **Tread** — lifetime distance on foot. The flagship, and the one with the best fiction attached: footfall is the Anvil's carrier signal, so walking is literally how the surface answers. Onyx sits around 10,000 km, which is years of ordinary life rather than a grind target.
  * **Practice** — collapses, translations, reforges, hearths visited, accessions, confluences recorded, uplinks sent. One line per verb the game has, so every playstyle ratchets something.
  * **Constancy** — total days played, seasons finished, and one keystone per civilization for codex completion. Those fifteen keystones double as the season spine and are the slow numbers that keep a four-year player moving.

Four rules that keep them healthy:

  1. **Distance only counts below the speed gate.** Car and train kilometres do not accrue. The medal and the safety rule in section 11 then agree instead of fighting, and most of the spoofing payoff evaporates with them.
  2. **No consecutive-day streaks.** This is the one place Ingress gets it wrong: Sojourner is what sends people out at 2 a.m. in the rain to protect a 900-day chain, and players describe it as an obligation rather than a game. Count _total_ days, or grant grace tokens that absorb a missed day automatically. Retention is nearly identical and nobody is walking to a dark car park at midnight over it.
  3. **Every distance medal has a non-distance sibling of equal prestige.** Distance itself is fine and inclusive — wheelchair users cover ground, and nothing here is speed-based — but a housebound player needs a Gilt of their own, so Practice and Constancy tiers must be reachable without leaving a 200 m radius.
  4. **Unlocked server-side from server-confirmed state.** Medals are effectively permanent; an exploit that grants Onyx cannot be quietly rolled back. Trust-flagged accounts still earn them but do not display on any social surface.

What stays out is the timed and the placed: a medal for playing between midnight and 4 a.m., anything rewarding arrival at a location before a deadline, anything that expires this week. Those create the pressure. A lifetime counter does not.

#### Leaderboards — the part that needs restraint

The obvious boards for a GPS game — distance, kills, captures — are precisely what spoofers farm, and a global distance board is unwinnable by anyone who has a job. Build boards that are _bounded_ and _knowledge-based_ instead:

  * **Seasonal codex completion** — bounded, so everyone can reach the ceiling and nobody grinds against a teenager with a rooted phone.
  * **Collection rarity index** — scores the improbability of what you hold, which rewards travel and confluence hunting rather than hours.
  * **Per-city and friends-only by default** , global opt-in. A board of 200 people in Lyon is played; a board of two million is scenery.
  * **No ranked distance, kill-count or time-to-anything boards.** Those quantities belong on medals, where they ratchet, not on a ladder that pits an unemployed twenty-year-old against a nurse on shifts.
  * **Submit from the server** , using the player's granted credential against the Play Games REST endpoint, never from the client, and gate every submission on the trust score in section 15 — flagged accounts submit successfully to a board only they can see. Verify the current API surface during the phase 0 spike; Google has moved these endpoints before.
  * Keep your own authoritative stats in Postgres regardless. Play Games boards reset, change shape, and cannot be queried the way your analytics need.

#### What you are giving up, honestly

No AR photo mode, which is the single most shareable thing the genre has ever produced, and the thing press always screenshots. Mitigate it with a 2D _plate_ mode: compose the monster or artefact over the device camera frame as a flat illustrated cut-out with a museum-label frame and the provenance line. It is cheap, it fits the catalogue aesthetic better than a floating 3D model does, and it gives people something to post.

You also give up an easy path to a console or PC version later. That is fine — this game does not have one.

iOS stays out of scope, but every game rule lives on the server and every civilization is data, so the iOS client is a client and not a second implementation.

## 13 · Server architecture

Built for 100k daily actives with headroom to ten times that, without paying for ten times that on day one.

_[Architecture diagram: client -> CDN + API gateway -> stateless game services (world, combat, player, beacons, season, trust) -> Redis + NATS -> Postgres/PostGIS -> ClickHouse; nightly offline world build from OSM + Wikidata feeds the cell weight store.]_

### Choices worth defending

  * **Go for the services.** Boring, cheap in memory per connection, excellent HTTP/2 and WebSocket story, easy hiring. Elixir is the better fit if you later want persistent per-cell processes with real-time multiplayer; Kotlin on the server is defensible if you want one language across client and back end. Pick one and do not mix three.
  * **Stateless services, sharded reads.** Because spawns are derived by hash and weights come from a read-only store, the world tier scales horizontally with no coordination. The only genuinely stateful things are the player record and the combat session, both short-lived and cheap.
  * **Postgres until it hurts.** One primary, read replicas, table partitioning on the event log. At 100k DAU this is comfortable. Do not start with a distributed database; start with a schema whose player table can be sharded by player id later.
  * **Region pinning.** Deploy in three regions (EU, US, APAC) with the player's home region pinned, and replicate the read-only world store to all three. A player who flies to another continent takes a small latency hit; the alternative is a global write path you do not need.

#### Load, roughly

At 100k DAU, assume 8% peak concurrency (8,000 sessions), one location update per 10 s and one game action per 20 s per active session: about **1,200 requests per second at peak**. That is three or four modest service nodes, one Redis cluster, one Postgres primary with two replicas. Tiles and art dwarf it in bytes and cost almost nothing on a CDN. Infrastructure at that scale lands somewhere around **$2–5k per month** ; it is not the expensive part of this project, and the plan should be sized for people, not servers.

## 14 · Data pipeline

  1. **Ingest.** Geofabrik regional extracts to start, full planet later, with minutely or daily diffs applied by osmium. Filter aggressively to the tag set in section 9 before loading — you need perhaps 0.5% of the planet file.
  2. **Classify.** Map tags to functions; normalise chains via `brand:wikidata`; resolve buildings to polygons; snap points off carriageways and out of private land.
  3. **Enrich.** Join museums to Wikidata and to curated holdings; attach civilization weight vectors; attach the manual blocklist and per-institution opt-outs.
  4. **Build cells.** Rasterise zones, corridors and beacons; run the fifteen cost-distance solves for the drift field over land, shelf and ocean graphs; normalise to top-four weights plus purity at H3 r6 and interpolate to r8; compute density normalisation factors; emit a versioned immutable snapshot.
  5. **Publish.** Snapshot to object storage; services hot-swap to the new version; the client gets a version header and invalidates local caches per region. Keep the previous snapshot for one week for instant rollback.
  6. **Moderate.** Player reports and institutional requests write to an override table that is applied on top of every build, so a takedown takes effect in minutes and survives the next rebuild.

Budget a full-time data engineer from month two. This pipeline is the game; content teams and designers will be blocked on it constantly, and a part-time owner will become the bottleneck.

## 15 · Anti-cheat

GPS spoofing is trivial on Android and there is a mature market of tools for it. You will not stop it; you can make it unprofitable and keep it out of the leaderboards.

  * **Play Integrity API** for device and app attestation; refuse rewards, but not play, on failed attestation.
  * **Mock location detection** plus checks on developer options, known spoofing packages, and rooted or hooked runtimes.
  * **Plausibility, server-side.** Speed between fixes, teleport distance over time, accuracy-radius patterns, straight-line movement, impossibly regular intervals, and the tell that catches most of them: GPS fixes that never wander when stationary.
  * **Sensor corroboration.** A walking session should produce step counts, barometric drift and accelerometer noise consistent with the claimed track. Sample it, do not stream it.
  * **Server authority over everything valuable.** Drops, accessions and translations are decided server-side from server state — and so are achievement unlocks and leaderboard submissions (section 12).
  * **No PC build.** Opting out of Google Play Games on PC removes the easiest spoofing platform from the board.
  * **Shadow enforcement.** Do not ban instantly. Score accounts, degrade rewards quietly, and sweep in batches — instant bans just teach the tool authors what your detection is.
  * **Museum abuse.** Require dwell time and a plausible approach track for an accession, not a single fix inside the polygon.

## 16 · Economy and monetization

  * **Season pass** , roughly $8–10 per 10-week chapter: cosmetic finishes, extra translation slots, extra vault rows, the chapter's cosmetic set. The main revenue line.
  * **Permanent convenience** : vault expansions, additional loadout presets, extra daily accession. One-time purchases, no subscription.
  * **Cosmetics** : patinas, display-case frames for provenance sets, map styles.
  * **No loot boxes, no gacha, no paid power.** Beyond the obvious, this keeps the door open with museums and schools, which is a strategic asset worth more than the revenue you would skim.
  * **Institutional** : a paid tier for museums that want gallery-level beacons, exhibition tie-ins and attendance analytics. Small in year one, plausibly significant in year three, and it is the line that makes this company interesting to anyone other than gamers.
  * **Sinks** : reforging consumes materials, translation consumes time, vault space is finite. The economy needs gear to leave circulation or the museum content stops mattering by month three.

## 17 · Legal and licensing

Area| What to do  
---|---  
OpenStreetMap (ODbL)| Attribute visibly. The share-alike question — whether your derived POI database triggers an obligation when you serve it to clients — is genuinely unsettled for games. Keep OSM-derived data in a separate, cleanly-bounded database from player data; be prepared to publish that derived database; get an opinion from a lawyer who has read the ODbL community guidelines, not a generalist.  
Patents| Niantic holds US patents around location-based gameplay. A freedom-to-operate review before you spend serious money is cheap insurance; design around anything that looks close.  
Museum names and marks| Nominative use of a museum's name is usually fine; logos are not. Get written permission before any branded tie-in, and honour opt-outs instantly.  
Object imagery| Use CC0 open-access imagery (the Met, Rijksmuseum, Smithsonian and others) or commission original art. Track the licence per asset in a register from day one, because retrofitting that is miserable.  
Privacy| Location is sensitive personal data under GDPR. Minimise, aggregate, and set short retention on raw tracks; publish a plain-language policy; no selling or sharing of location data, ever, and say so loudly — it is a differentiator.  
Children| The game will attract minors. Age gate, GDPR-K and COPPA compliance, no open chat at launch, and no location-revealing social features.  
Sanctions and access| Several core zones sit in sanctioned or conflict regions. Decide deliberately how the app behaves in Iraq, Syria, Ukraine and Sudan, including whether it should do anything at all in an active conflict zone.  
  
## 18 · Cultural guardrails

The premise — vanished civilizations, a hidden society that erased them, a hollow earth — sits two steps from a genre of real conspiracy theory with an ugly history. Walk those two steps deliberately, in the other direction.

  * **The Concord is not a real organisation and must never resemble one.** No ancient bloodlines, no bankers, no single ethnicity, no nod to any real secret-society mythology. Write it as what it is: a cartel of archivists and administrators, boring, bureaucratic, multinational, recruiting from everywhere. Boring is the safeguard.
  * **Descendants exist.** Maya, Egyptian, Nigerian, Peruvian, Chinese and Cretan communities are living, present-tense peoples. The fiction should be careful that "they went below" reads as a metaphor for erasure rather than as extinction — and the game's non-fiction lore cards should say plainly who the descendant communities are.
  * **Hire consultants, not just researchers.** One archaeologist or historian per civilization for a review pass, paid, credited in the app. It costs perhaps €30–50k across the roster and it is the difference between a game museums promote and one they send lawyers to.
  * **Separate fact from fiction visually and always.** If a player cannot tell which half of a lore card is invented, you have made a misinformation machine with a loot loop attached.
  * **No treasure hunting.** Never imply that real digging, detecting or collecting is part of the game. Put an explicit anti-looting line in the onboarding and mean it.

## 19 · Roadmap, team, budget

Phase| Length| Scope| Gate to pass  
---|---|---|---  
0 · Prove the spine| 5 weeks| Throwaway prototype: one city's OSM extract, H3 weights for three civilizations, a placeholder fight, real drops. No art.| Does walking a real route through real zones feel like anything? If not, stop here.  
1 · Vertical slice| 3 months| Etruria, Kemet, Hopewell. One metro area plus 20 curated museums worldwide. Full combat, gear, vault, hearth, wardrobe, day/night and the condition vector, Play Games sign-in and the account model. Final art direction on 30 items.| 20 external testers, 5-day retention above 35%.  
2 · Closed beta| 3 months| All fifteen, one continent of soil data, top 150 museums. Codex, chapter one, drift hybrids, weather and the spawn rule language, passive collection, anti-cheat, medals and leaderboards, safety blocklist.| Crash-free above 99.5%; battery under 6% per 30 min session; no safety incidents.  
3 · Soft launch| 3 months| Two or three countries on Play. Live ops, economy tuning, the event framework, the first museum partnership.| D30 retention above 12%, ARPDAU sane, infrastructure cost per DAU under control.  
4 · Global| —| Planet data, season two, the Concord turn, iOS client begins.| —  
  
#### Team through phase 3 (about nine people)

Game director and designer · two Android engineers · two back-end engineers · one data engineer (geo) · one artist plus contract illustrators · one writer and researcher · one producer who also owns community and institutional relations. Add part-time: a DevOps contractor, a lawyer, archaeology consultants per civilization.

#### Money, order of magnitude

  * Phases 0–3, roughly 14 months of a nine-person team in Western Europe: **€1.1–1.6M** fully loaded.
  * Art and audio contract work: **€150–250k**.
  * Research, consultants, curation of the museum table: **€60–100k**.
  * Infrastructure through soft launch: under **€40k**.
  * Legal, including the freedom-to-operate review: **€40–80k**.

A leaner path exists — three or four people, 18 months, one civilization trio, museums restricted to one country — and it is a reasonable way to reach phase 1 on a fraction of that before raising anything.

## 20 · Risk register

Risk| Severity| Response  
---|---|---  
A safety or dignity incident at a sensitive site| Existential| Blocklist and human review before launch; instant takedown path; a named person responsible  
The walking loop is not actually fun without a beloved IP behind it| High| Phase 0 exists precisely to find this out for €60k rather than €1.5M  
OSM data quality varies wildly by country| High| Density normalisation, synthesised cairns, and a country-readiness score gating launches  
Museum beacons get abused or an institution objects publicly| Medium| Opt-out switch, quiet mode in galleries, partnership outreach before launch, not after  
Drift hybrids read as invented artefacts and muddy the educational claim| Medium| No non-fiction panel on hybrid cards, explicit "made by the transmission" line, both source objects linked with real cards  
Spoofing floods the economy| Medium| Server authority, shadow enforcement, no player trading at launch  
Public leaderboards become the spoofers' scoreboard and the honest players leave| Medium| Bounded, knowledge-based boards only; city and friend scope by default; trust-gated shadow submission; volume quantities live on medals instead  
A weather provider outage flattens the world mid-season| Low| Stale-while-revalidate, fair-weather fallback, a paid second provider from soft launch  
Condition-gated content is unobtainable at some latitudes| Medium| Anomaly-based triggers against local normals; a second acquisition path for every gated item  
Medals become obligation and push players out at unsafe hours| Medium| No consecutive-day streaks, no timed or placed medals, distance counted only below the speed gate  
Battery drain kills retention| Medium| Treat battery as a tracked metric with a hard budget from phase 1, not an optimisation pass at the end  
Patent or ODbL challenge| Medium| Freedom-to-operate review early; clean separation of the derived database  
The conspiracy premise is read as endorsing real conspiracy theory| Medium| Section 18, enforced by a writer with a veto  
Content volume: fifteen civilizations is fifteen art and writing budgets| High| Shared silhouettes and rigs, civilization skinned by palette, pattern and prop; three deep at launch, twelve broad  
  
## 21 · Decisions still open

Three are now closed: fifteen civilizations with full planetary coverage, museums as a feature rather than the thesis, 2D native Android. What is left:

  1. **Which three civilizations ship deep in phase 1?** My pick is Etruria, Kemet and Hopewell — one that rewards a European old town, one that every major museum on earth holds, and one that makes the United States work on day one. Say if you want a different trio; it sets the art direction.
  2. **Which city is home?** Phase 0 needs one metro area with dense OSM coverage, real ancient soil, and a museum you can walk into repeatedly. Rome, Bologna, Paris, London and Athens all qualify for different reasons.
  3. **Who is on the team today?** The budget in section 19 assumes nine people. If this is you and one collaborator, I will rewrite phases 0 and 1 for that and cut hard.
  4. **How central is travel?** There is a version built entirely around the trip — a passport metaphor, much stronger distance rewards, fifteen stamps to collect. Smaller audience, far more distinctive product.
  5. **Social at launch, or not?** I have assumed single-player. Co-located co-op is the natural first step and also a whole moderation surface.
  6. **Australia, and the descendant-community policy.** Section 2 lays out three options and recommends the silent zone. This needs a real answer before any map data ships, not after.

Next useful artefacts, on request: the phase 0 technical spike plan week by week; the H3 weight builder as working code against a real Geofabrik extract; the Wikidata SPARQL query for the museum holdings table; a combat maths model with sample loadouts; or a ten-slide investor version of this document.

Map data plans in this document assume OpenStreetMap, © OpenStreetMap contributors, ODbL.
