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
    /// calls are Coordinates.convertCoordinateToVector3() and the LocationManager's current
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
                return c.convertCoordinateToVector3();
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

        /// <summary>The map's own idea of where the player is, when it has one.</summary>
        public bool TryGetMapLocation(out double lat, out double lon)
        {
            lat = lon = 0;
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
