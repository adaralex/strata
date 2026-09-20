using System.Collections.Generic;
using Newtonsoft.Json;

// Mirrors of worldd's JSON (server/cmd/worldd/main.go). The client only reads these; it never
// tells the server what it looted.
namespace Strata.Net
{
    public sealed class CivWeight
    {
        [JsonProperty("civ")] public string Civ;
        [JsonProperty("weight")] public double Weight;
    }

    public sealed class BeaconOut
    {
        [JsonProperty("ref")] public string Ref;
        [JsonProperty("name")] public string Name;
        [JsonProperty("grade")] public int Grade;
        [JsonProperty("civs")] public Dictionary<string, double> Civs;
        [JsonProperty("civ_source")] public string Source;
        [JsonProperty("distance_m")] public double DistanceM;
    }

    public sealed class Zone
    {
        [JsonProperty("kind")] public string Kind;
        [JsonProperty("ref")] public string Ref;
        [JsonProperty("name")] public string Name;
        [JsonProperty("buffer_m")] public double BufferM;
    }

    public sealed class Cond
    {
        [JsonProperty("epoch")] public long Epoch;
        [JsonProperty("phase")] public string Phase;
        [JsonProperty("is_night")] public bool IsNight;
        [JsonProperty("solar_alt_deg")] public double SolarAltDeg;
        [JsonProperty("moon_illumination")] public double MoonIllumination;
        [JsonProperty("propagation")] public double Propagation;
        [JsonProperty("after_midnight")] public bool AfterMidnight;
        [JsonProperty("weather")] public string Weather;
    }

    public sealed class LookupOut
    {
        [JsonProperty("cell")] public string Cell;
        [JsonProperty("found")] public bool Found;
        [JsonProperty("propagation")] public double Propagation;
        [JsonProperty("weights")] public List<CivWeight> Weights = new List<CivWeight>();
        [JsonProperty("residual")] public double Residual;
        [JsonProperty("purity")] public double Purity;
        [JsonProperty("services")] public List<string> Services = new List<string>();
        [JsonProperty("beacons")] public List<BeaconOut> Beacons = new List<BeaconOut>();
        [JsonProperty("beacon_grade")] public int BeaconGrade;
        [JsonProperty("excluded")] public bool Excluded;
        [JsonProperty("zone")] public Zone Zone;
        [JsonProperty("excluded_cell")] public bool ExcludedCell;
        [JsonProperty("wild")] public bool Wild;
        [JsonProperty("water_band")] public int WaterBand;
        [JsonProperty("urban_band")] public int UrbanBand;
        [JsonProperty("cond")] public Cond Cond;

        public string DominantCiv => Weights.Count > 0 ? Weights[0].Civ : null;
    }

    public sealed class Spawn
    {
        [JsonProperty("id")] public string Id;
        [JsonProperty("cell")] public string Cell;
        [JsonProperty("epoch")] public long Epoch;
        [JsonProperty("expires_at")] public long ExpiresAt;
        [JsonProperty("civ")] public string Civ;
        [JsonProperty("secondary")] public string Secondary;
        [JsonProperty("kind")] public string Kind;
        [JsonProperty("name")] public string Name;
        [JsonProperty("tags")] public List<string> Tags;
        [JsonProperty("fiction")] public string Fiction;
        [JsonProperty("rank")] public string Rank;
        [JsonProperty("tier")] public int Tier;
        [JsonProperty("tier_name")] public string TierName;
        [JsonProperty("authenticity")] public string Authenticity;
        [JsonProperty("value_mul")] public double ValueMul;
        [JsonProperty("lat")] public double Lat;
        [JsonProperty("lon")] public double Lon;
        [JsonProperty("distance_m")] public double DistanceM;
    }

    public sealed class SpawnsOut
    {
        [JsonProperty("cond")] public Cond Cond;
        [JsonProperty("epoch")] public long Epoch;
        [JsonProperty("epoch_ends")] public long EpochEnds;
        [JsonProperty("sense_range_m")] public double SenseRangeM;
        [JsonProperty("speed_gated")] public bool SpeedGated;
        [JsonProperty("spawns")] public List<Spawn> Spawns = new List<Spawn>();
    }

    public sealed class Affix
    {
        [JsonProperty("k")] public string K;
        [JsonProperty("v")] public double V;
    }

    public sealed class Provenance
    {
        [JsonProperty("kind")] public string Kind;
        [JsonProperty("ref")] public string Ref;
        [JsonProperty("name")] public string Name;
        [JsonProperty("cell")] public string Cell;
        [JsonProperty("acquired_at")] public string AcquiredAt;
    }

    public sealed class Buff
    {
        [JsonProperty("stat")] public string Stat;
        [JsonProperty("pct")] public double Pct;
        [JsonProperty("minutes")] public int Minutes;
    }

    public sealed class Edible
    {
        [JsonProperty("id")] public string Id;
        [JsonProperty("name")] public string Name;
        [JsonProperty("buff")] public Buff Buff;
    }

    public sealed class Item
    {
        [JsonProperty("id")] public string Id;
        [JsonProperty("kind")] public string Kind;
        [JsonProperty("civ")] public string Civ;
        [JsonProperty("secondary")] public string Secondary;
        [JsonProperty("archetype")] public string Archetype;
        [JsonProperty("slot")] public string Slot;
        [JsonProperty("name")] public string Name;
        [JsonProperty("tier")] public int Tier;
        [JsonProperty("tier_name")] public string TierName;
        [JsonProperty("authenticity")] public string Authenticity;
        [JsonProperty("affixes")] public List<Affix> Affixes;
        [JsonProperty("provenance")] public Provenance Provenance;
        [JsonProperty("fact")] public string Fact;
        [JsonProperty("fiction")] public string Fiction;
        [JsonProperty("edible")] public Edible Edible;
        [JsonProperty("codex_ref")] public string CodexRef;

        public bool IsHybrid => !string.IsNullOrEmpty(Secondary);
    }

    public sealed class CollapseResult
    {
        [JsonProperty("spawn")] public Spawn Spawn;
        [JsonProperty("item")] public Item Item;
        [JsonProperty("beacon")] public BeaconOut Beacon;
    }

    public sealed class CollapseIn
    {
        [JsonProperty("spawn_id")] public string SpawnId;
        [JsonProperty("cell")] public string Cell;
        [JsonProperty("epoch")] public long Epoch;
        [JsonProperty("lat")] public double Lat;
        [JsonProperty("lon")] public double Lon;
    }

    public sealed class ErrorOut
    {
        [JsonProperty("error")] public string Error;
    }
}
