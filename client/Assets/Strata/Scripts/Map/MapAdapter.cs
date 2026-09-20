using System;
using System.Collections.Generic;
using UnityEngine;
#if STRATA_GOMAP
using GoMap;
using GoShared;
#endif

namespace Strata.Map
{
    /// <summary>
    /// The only file that names GO Map. Two jobs: turn a lat/lon into a world position on
    /// the map, and expose the map's own location when the device has none (the editor's
    /// GO Map simulation). Add STRATA_GOMAP to Scripting Define Symbols once the asset is
    /// imported; without it a flat local plane stands in, so the scene runs in the editor
    /// before GO Map is set up.
    ///
    /// If your GO Map version names things differently, this is the file to fix: the two
    /// calls are Coordinates.convertCoordinateToVector() and the LocationManager's current
    /// location.
    /// </summary>
    public sealed class MapAdapter : MonoBehaviour
    {
#if STRATA_GOMAP
        [Tooltip("The GOMap component on the map object from the GO Map prefab.")]
        public GOMap goMap;
#endif
        [Tooltip("Fallback plane: metres per Unity unit when GO Map is not compiled in.")]
        public float fallbackMetresPerUnit = 1f;

        private bool _originSet;
        private double _originLat, _originLon;

        public Vector3 WorldPosition(double lat, double lon)
        {
#if STRATA_GOMAP
            if (goMap != null)
            {
                var c = new Coordinates(lat, lon, 0);
                return c.convertCoordinateToVector();
            }
#endif
            if (!_originSet)
            {
                _originSet = true;
                _originLat = lat;
                _originLon = lon;
            }
            double rad = System.Math.PI / 180;
            double x = (lon - _originLon) * rad * 6371008.8 * System.Math.Cos(_originLat * rad);
            double z = (lat - _originLat) * rad * 6371008.8;
            return new Vector3((float)(x / fallbackMetresPerUnit), 0f, (float)(z / fallbackMetresPerUnit));
        }

        private bool _simSet;
        private double _simLat, _simLon, _simReloadLat, _simReloadLon;

        /// <summary>
        /// Editor emulation: put the map's simulated location here. With GO Map this drives its
        /// location manager and asks it to reload tiles every 60 m; without it the value is
        /// what TryGetMapLocation returns.
        /// </summary>
        public void SetSimulatedLocation(double lat, double lon)
        {
            _simSet = true;
            _simLat = lat; _simLon = lon;
#if STRATA_GOMAP
            if (goMap != null && goMap.locationManager != null)
            {
                var c = new Coordinates(lat, lon, 0);
                goMap.locationManager.currentLocation = c;
                if (Geo.DistanceM(_simReloadLat, _simReloadLon, lat, lon) > 60)
                {
                    _simReloadLat = lat; _simReloadLon = lon;
                    goMap.locationManager.onLocationChanged?.Invoke(c);
                }
            }
#endif
        }

        /// <summary>The map's own idea of where the player is, when it has one.</summary>
        public bool TryGetMapLocation(out double lat, out double lon)
        {
            lat = lon = 0;
            if (_simSet) { lat = _simLat; lon = _simLon; return true; }
#if STRATA_GOMAP
            if (goMap != null && goMap.locationManager != null)
            {
                var c = goMap.locationManager.currentLocation;
                if (c != null && (c.latitude != 0 || c.longitude != 0))
                {
                    lat = c.latitude;
                    lon = c.longitude;
                    return true;
                }
            }
#endif
            return false;
        }

        /// <summary>
        /// Route the map's point-of-interest layer to service markers: for each service, the
        /// tile kinds that show it; a template to clone per service; and a callback with the
        /// service, the point's name and the placed clone. Without GO Map this is a no-op.
        /// </summary>
        public void ConfigureServices(IReadOnlyDictionary<string, string[]> kindsByService, Func<string, GameObject> templateFor, Action<string, string, GameObject> onPlaced)
        {
#if STRATA_GOMAP
            if (goMap == null || goMap.pois == null) return;
            var list = new List<GOPOIRendering>();
            foreach (var kv in kindsByService)
            {
                var service = kv.Key;
                var template = templateFor(service);
                foreach (var kindName in kv.Value)
                {
                    if (!Enum.TryParse(kindName, true, out GOPOIKind kind)) { Debug.LogWarning($"MapAdapter: no POI kind '{kindName}' for {service}"); continue; }
                    var r = new GOPOIRendering { kind = kind, prefab = template, tag = "", OnPoiLoad = new GOFeatureEvent() };
                    r.OnPoiLoad.AddListener((feature, clone) => onPlaced(service, feature != null ? feature.name : null, clone));
                    list.Add(r);
                }
            }
            goMap.pois.renderingOptions = list.ToArray();
            goMap.pois.disabled = false;
#endif
        }

        public bool UsingGoMap
        {
            get
            {
#if STRATA_GOMAP
                return goMap != null;
#else
                return false;
#endif
            }
        }
    }
}
