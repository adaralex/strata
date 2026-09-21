using System;
using System.Collections.Generic;
using System.IO;
using Newtonsoft.Json;
using UnityEngine;
using Strata.Net;

namespace Strata.Play
{
    /// <summary>
    /// What this phone has been given and what it has fought, kept between walks in the
    /// app's persistent data path. Display state only: every item here is exactly what
    /// worldd returned from a collapse, and equipping is a choice of what to show, not a
    /// claim the server has to honour (non-negotiable 1). Stats are the plan's four
    /// (PLAN §8: Might, Ward, Resonance, Fortune), summed from the affixes of equipped gear.
    /// </summary>
    public sealed class PlayerStore
    {
        public const string FileName = "strata-player.json";
        public static readonly string[] Slots = { "main", "off", "body", "head", "charm1", "charm2", "relic" };
        public static readonly string[] Stats = { "might", "ward", "resonance", "fortune" };

        public sealed class CodexEntry
        {
            public string kind, name, civ, secondary, rank, tierName, fiction;
            public List<string> tags;
            public int fought, won, lost, autoResolved;
            public string firstAt, lastAt;
        }

        public sealed class Battle
        {
            public int fights, wins, losses, autoResolves, drops;
            public Dictionary<string, int> winsByCiv = new Dictionary<string, int>();
        }

        public sealed class FightPoint
        {
            public double lat, lon;
            public string civ, name, rank;
            public bool won;
            public string at;
        }

        public string createdAt;
        public int walks;
        public double totalMetres;
        public Dictionary<string, double> metresByCiv = new Dictionary<string, double>();
        public List<FightPoint> fightPoints = new List<FightPoint>();
        public List<Item> items = new List<Item>();
        public Dictionary<string, string> equipped = new Dictionary<string, string>(); // slot -> item id
        public Dictionary<string, CodexEntry> codex = new Dictionary<string, CodexEntry>(); // spawn kind -> entry
        public Battle battle = new Battle();

        public event Action Changed; // events are never serialised

        private static string FilePath => Path.Combine(Application.persistentDataPath, FileName);

        public static PlayerStore Load()
        {
            try
            {
                if (File.Exists(FilePath))
                {
                    var s = JsonConvert.DeserializeObject<PlayerStore>(File.ReadAllText(FilePath));
                    if (s != null) return s.Normalise();
                }
            }
            catch (Exception e) { Debug.LogWarning("player store unreadable, starting fresh: " + e.Message); }
            return new PlayerStore { createdAt = DateTimeOffset.UtcNow.ToString("o") };
        }

        private PlayerStore Normalise()
        {
            items = items ?? new List<Item>();
            equipped = equipped ?? new Dictionary<string, string>();
            codex = codex ?? new Dictionary<string, CodexEntry>();
            battle = battle ?? new Battle();
            battle.winsByCiv = battle.winsByCiv ?? new Dictionary<string, int>();
            metresByCiv = metresByCiv ?? new Dictionary<string, double>();
            fightPoints = fightPoints ?? new List<FightPoint>();
            return this;
        }

        // ---- distance and walks ----

        private double _unsavedMetres;

        /// <summary>Accrue distance under a civilization; saved by SaveIfDue or the next Save.</summary>
        public void AddMetres(string civ, double metres)
        {
            if (metres <= 0) return;
            civ = string.IsNullOrEmpty(civ) ? "unknown" : civ;
            metresByCiv.TryGetValue(civ, out var m);
            metresByCiv[civ] = m + metres;
            totalMetres += metres;
            _unsavedMetres += metres;
        }

        /// <summary>Write the file when at least this much distance is unsaved.</summary>
        public void SaveIfDue(double thresholdMetres = 100)
        {
            if (_unsavedMetres >= thresholdMetres) { _unsavedMetres = 0; Save(); }
        }

        public void EndWalk()
        {
            walks++;
            _unsavedMetres = 0;
            Save();
        }

        /// <summary>The civilization this walker leans to: most wins, then most ground walked.</summary>
        public string PreferredCiv()
        {
            string best = null; int bestWins = -1; double bestM = -1;
            foreach (var kv in battle.winsByCiv)
            {
                metresByCiv.TryGetValue(kv.Key, out var m);
                if (kv.Value > bestWins || (kv.Value == bestWins && m > bestM)) { best = kv.Key; bestWins = kv.Value; bestM = m; }
            }
            if (best != null) return best;
            foreach (var kv in metresByCiv) if (kv.Key != "unknown" && kv.Value > bestM) { best = kv.Key; bestM = kv.Value; }
            return best;
        }

        /// <summary>Civilizations the worn gear represents, most pieces first.</summary>
        public List<KeyValuePair<string, int>> GearCivs()
        {
            var counts = new Dictionary<string, int>();
            foreach (var slot in Slots)
            {
                var it = EquippedIn(slot);
                if (it?.Civ == null) continue;
                counts.TryGetValue(it.Civ, out var c);
                counts[it.Civ] = c + 1;
            }
            var list = new List<KeyValuePair<string, int>>(counts);
            list.Sort((a, b) => b.Value.CompareTo(a.Value));
            return list;
        }

        // ---- achievements: derived, never stored ----

        public sealed class Achievement
        {
            public string Id, Title, Detail;
            public bool Earned;
        }

        public List<Achievement> Achievements()
        {
            int wornSlots = 0; foreach (var s in Slots) if (EquippedIn(s) != null) wornSlots++;
            var civsHeld = new HashSet<string>(); foreach (var i in items) if (i.Civ != null) civsHeld.Add(i.Civ);
            bool hybrid = items.Exists(i => i.IsHybrid), grounded = items.Exists(i => i.Authenticity == "grounded");
            bool elite = false, boss = false; foreach (var e in codex.Values) { if (e.won > 0 && e.rank == "elite") elite = true; if (e.won > 0 && e.rank == "boss") boss = true; }
            int tiers = 0; foreach (var i in items) if (i.Tier >= 2) tiers++;
            return new List<Achievement>
            {
                A("first_blood", "First Blood", "Win a fight", battle.wins >= 1),
                A("ten_fights", "Blooded", "Win ten fights", battle.wins >= 10),
                A("fifty_fights", "Veteran", "Win fifty fights", battle.wins >= 50),
                A("first_drop", "Something Followed", "Receive a drop", battle.drops >= 1),
                A("grounded", "Of This Soil", "A grounded item", grounded),
                A("hybrid", "Made By The Transmission", "A drift hybrid", hybrid),
                A("votive", "Votive", "An item of votive tier or above", tiers >= 1),
                A("elite", "Elite Slayer", "Beat an elite", elite),
                A("boss", "Area Boss", "Beat an area boss", boss),
                A("km1", "First Kilometre", "Walk 1 km", totalMetres >= 1000),
                A("km5", "Five Kilometres", "Walk 5 km", totalMetres >= 5000),
                A("km10", "Ten Kilometres", "Walk 10 km", totalMetres >= 10000),
                A("km25", "Twenty-Five", "Walk 25 km", totalMetres >= 25000),
                A("four_civs", "Four Peoples", "Items from four civilizations", civsHeld.Count >= 4),
                A("full_kit", "Fully Dressed", "Every slot worn", wornSlots >= Slots.Length),
                A("codex10", "Ten Names", "Ten monsters in the codex", codex.Count >= 10),
                A("codex25", "Bestiary", "Twenty-five monsters in the codex", codex.Count >= 25),
                A("walks3", "Habit", "Three walks ended", walks >= 3),
            };
        }

        private static Achievement A(string id, string title, string detail, bool earned) => new Achievement { Id = id, Title = title, Detail = detail, Earned = earned };

        public void Save()
        {
            try { File.WriteAllText(FilePath, JsonConvert.SerializeObject(this, Formatting.Indented)); }
            catch (Exception e) { Debug.LogWarning("player store save failed: " + e.Message); }
            Changed?.Invoke();
        }

        // ---- items and equipment ----

        public void AddItem(Item item)
        {
            if (item == null) return;
            if (string.IsNullOrEmpty(item.Id)) item.Id = Guid.NewGuid().ToString("N");
            if (item.Provenance != null && string.IsNullOrEmpty(item.Provenance.AcquiredAt)) item.Provenance.AcquiredAt = DateTimeOffset.UtcNow.ToString("o");
            items.Add(item);
            battle.drops++;
            Save();
        }

        public Item Find(string id) => string.IsNullOrEmpty(id) ? null : items.Find(i => i.Id == id);

        public Item EquippedIn(string slot) => equipped.TryGetValue(slot, out var id) ? Find(id) : null;

        public bool IsEquipped(Item item) => item != null && equipped.ContainsValue(item.Id);

        public bool CanEquip(Item item) => SlotFor(item) != null;

        /// <summary>The equipment slot this item goes to; charms take the first free of two.</summary>
        public string SlotFor(Item item)
        {
            if (item == null || item.Kind == "edible" || string.IsNullOrEmpty(item.Slot)) return null;
            if (item.Slot == "charm")
            {
                foreach (var kv in equipped) if (kv.Value == item.Id) return kv.Key;
                if (!equipped.ContainsKey("charm1")) return "charm1";
                if (!equipped.ContainsKey("charm2")) return "charm2";
                return "charm1";
            }
            return Array.IndexOf(Slots, item.Slot) >= 0 ? item.Slot : null;
        }

        public void Equip(Item item)
        {
            var slot = SlotFor(item);
            if (slot == null) return;
            UnequipQuiet(item);
            equipped[slot] = item.Id;
            Save();
        }

        public void Unequip(Item item)
        {
            if (UnequipQuiet(item)) Save();
        }

        private bool UnequipQuiet(Item item)
        {
            var keys = new List<string>();
            foreach (var kv in equipped) if (kv.Value == item.Id) keys.Add(kv.Key);
            foreach (var k in keys) equipped.Remove(k);
            return keys.Count > 0;
        }

        public void UnequipSlot(string slot)
        {
            if (equipped.Remove(slot)) Save();
        }

        /// <summary>Items that could go in a slot; charm1 and charm2 both take charms.</summary>
        public List<Item> ItemsForSlot(string slot)
        {
            var key = slot.StartsWith("charm") ? "charm" : slot;
            var list = new List<Item>();
            foreach (var i in items) if (i.Kind != "edible" && i.Slot == key) list.Add(i);
            return list;
        }

        public Dictionary<string, double> StatTotals()
        {
            var t = new Dictionary<string, double>();
            foreach (var s in Stats) t[s] = 0;
            foreach (var slot in Slots)
            {
                var it = EquippedIn(slot);
                if (it?.Affixes == null) continue;
                foreach (var a in it.Affixes)
                {
                    t.TryGetValue(a.K, out var v);
                    t[a.K] = v + a.V;
                }
            }
            return t;
        }

        // ---- codex and battle record ----

        public void RecordFight(Spawn sp, bool won, bool auto, double? lat = null, double? lon = null)
        {
            battle.fights++;
            if (lat.HasValue && lon.HasValue)
                fightPoints.Add(new FightPoint { lat = lat.Value, lon = lon.Value, civ = sp?.Civ, name = sp?.Name, rank = sp?.Rank, won = won, at = DateTimeOffset.UtcNow.ToString("o") });
            if (won) battle.wins++; else battle.losses++;
            if (auto) battle.autoResolves++;
            if (sp != null)
            {
                var civ = sp.Civ ?? "?";
                if (won)
                {
                    battle.winsByCiv.TryGetValue(civ, out var w);
                    battle.winsByCiv[civ] = w + 1;
                }
                var key = string.IsNullOrEmpty(sp.Kind) ? sp.Name : sp.Kind;
                if (!codex.TryGetValue(key, out var e))
                {
                    e = new CodexEntry { kind = key, firstAt = DateTimeOffset.UtcNow.ToString("o") };
                    codex[key] = e;
                }
                e.name = sp.Name; e.civ = sp.Civ; e.secondary = sp.Secondary; e.rank = sp.Rank; e.tierName = sp.TierName; e.fiction = sp.Fiction; e.tags = sp.Tags;
                e.fought++;
                if (won) e.won++; else e.lost++;
                if (auto) e.autoResolved++;
                e.lastAt = DateTimeOffset.UtcNow.ToString("o");
            }
            Save();
        }

        public List<CodexEntry> CodexByRecency()
        {
            var list = new List<CodexEntry>(codex.Values);
            list.Sort((a, b) => string.CompareOrdinal(b.lastAt, a.lastAt));
            return list;
        }

        public List<Item> ItemsByRecency()
        {
            var list = new List<Item>(items);
            list.Reverse();
            return list;
        }
    }
}
