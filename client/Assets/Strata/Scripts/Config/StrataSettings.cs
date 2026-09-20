using System;
using UnityEngine;

namespace Strata.Config
{
    /// <summary>
    /// Runtime settings for the walk test. Create one via Assets > Create > Strata > Settings
    /// and assign it to WalkController, or leave it empty: defaults apply and the base URL is
    /// editable from the in-app settings panel. The device id is an anonymous GUID kept in
    /// PlayerPrefs; it is not identity (non-negotiable 1).
    /// </summary>
    [CreateAssetMenu(menuName = "Strata/Settings", fileName = "StrataSettings")]
    public sealed class StrataSettings : ScriptableObject
    {
        [Tooltip("worldd base URL, e.g. http://192.168.1.20:8080 for a laptop on the same hotspot.")]
        public string baseUrl = "http://192.168.1.20:8080";

        [Tooltip("Seconds between location polls.")]
        public float fixIntervalSeconds = 2f;

        [Tooltip("Refresh spawns after moving this far, metres.")]
        public float spawnRefreshDistanceM = 50f;

        [Tooltip("Refresh lookup and spawns at least this often, seconds.")]
        public float refreshIntervalSeconds = 30f;

        [Tooltip("Interaction range in metres. The server enforces its own; this only decides when a tap is offered.")]
        public float interactionRangeM = 40f;

        private const string BaseUrlKey = "strata.baseUrl";
        private const string DeviceKey = "strata.device";

        /// <summary>Base URL, overridden by the in-app setting when one was saved.</summary>
        public string EffectiveBaseUrl
        {
            get
            {
                var saved = PlayerPrefs.GetString(BaseUrlKey, "");
                return string.IsNullOrWhiteSpace(saved) ? baseUrl.TrimEnd('/') : saved.TrimEnd('/');
            }
        }

        public void SaveBaseUrl(string url)
        {
            PlayerPrefs.SetString(BaseUrlKey, url.Trim());
            PlayerPrefs.Save();
        }

        /// <summary>Anonymous per-install id, generated once.</summary>
        public static string DeviceId
        {
            get
            {
                var id = PlayerPrefs.GetString(DeviceKey, "");
                if (string.IsNullOrEmpty(id))
                {
                    id = "walk-" + Guid.NewGuid().ToString("N");
                    PlayerPrefs.SetString(DeviceKey, id);
                    PlayerPrefs.Save();
                }
                return id;
            }
        }

        public static StrataSettings Defaults()
        {
            var s = CreateInstance<StrataSettings>();
            s.name = "StrataSettings (runtime defaults)";
            return s;
        }
    }
}
