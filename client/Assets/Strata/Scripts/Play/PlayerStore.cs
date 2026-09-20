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

        public string createdAt;
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
            return this;
        }

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

        public void RecordFight(Spawn sp, bool won, bool auto)
        {
            battle.fights++;
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
