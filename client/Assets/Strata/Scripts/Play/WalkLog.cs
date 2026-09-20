using System;
using System.Collections.Generic;
using Strata.Map;
using Strata.Net;

namespace Strata.Play
{
    /// <summary>
    /// Distance walked under each dominant civilization, and the items received. This is
    /// PLAN §3's "you walked through 2.1 km of Hallstatt ground" with no background service.
    /// Pure logic, no Unity, so it is trivially testable.
    /// </summary>
    [Serializable]
    public sealed class WalkLog
    {
        public string device;
        public string startedAt = DateTimeOffset.UtcNow.ToString("o");
        public string endedAt;
        public Dictionary<string, double> metresByCiv = new Dictionary<string, double>();
        public double totalMetres;
        public int fights, wins, autoResolves;
        public List<ItemLine> items = new List<ItemLine>();
        public List<string> cellsVisited = new List<string>();

        [Serializable]
        public sealed class ItemLine
        {
            public string name, civ, secondary, tier, authenticity, kind, provenance;
            public string at;
        }

        private double? _lastLat, _lastLon;
        private string _currentCiv;

        public void SetGround(LookupOut lk)
        {
            _currentCiv = lk?.DominantCiv;
            if (lk != null && lk.Found && !cellsVisited.Contains(lk.Cell)) cellsVisited.Add(lk.Cell);
        }

        /// <summary>Accrue distance from the last fix. Jumps over 200 m (GPS glitch) are ignored.</summary>
        public void Fix(double lat, double lon)
        {
            if (_lastLat.HasValue)
            {
                var d = Geo.DistanceM(_lastLat.Value, _lastLon.Value, lat, lon);
                if (d < 200)
                {
                    totalMetres += d;
                    var civ = _currentCiv ?? "unknown";
                    metresByCiv.TryGetValue(civ, out var m);
                    metresByCiv[civ] = m + d;
                }
            }
            _lastLat = lat;
            _lastLon = lon;
        }

        public void Fought(bool won, bool auto)
        {
            fights++;
            if (won) wins++;
            if (auto) autoResolves++;
        }

        public void Received(Item item)
        {
            items.Add(new ItemLine
            {
                name = item.Name, civ = item.Civ, secondary = item.Secondary, tier = item.TierName,
                authenticity = item.Authenticity, kind = item.Kind,
                provenance = item.Provenance?.Kind + (string.IsNullOrEmpty(item.Provenance?.Name) ? "" : " " + item.Provenance.Name),
                at = DateTimeOffset.UtcNow.ToString("o"),
            });
        }

        public string Summary()
        {
            var lines = new List<string>();
            var sorted = new List<KeyValuePair<string, double>>(metresByCiv);
            sorted.Sort((a, b) => b.Value.CompareTo(a.Value));
            foreach (var kv in sorted)
                lines.Add($"{kv.Value / 1000:0.0} km of {UI.CivPalette.Label(kv.Key)} ground");
            lines.Add($"{items.Count} thing{(items.Count == 1 ? "" : "s")} followed you home, {wins}/{fights} fights won");
            return string.Join("\n", lines);
        }
    }
}
