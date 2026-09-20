#if UNITY_EDITOR
using System.Collections.Generic;
using Newtonsoft.Json;
using UnityEngine;
using Strata.Map;

namespace Strata.Play
{
    /// <summary>
    /// Editor-only walk emulation: moves the map's simulated location along a recorded route
    /// at walking speed so the real client loop (fixes, lookups, spawns, fights, drops, the
    /// walk log) runs as it would on the street. Optionally engages the nearest spawn in
    /// range and auto-resolves the fight, the way a tester tapping everything would. Never
    /// compiled into the phone build.
    /// </summary>
    public sealed class WalkSimulator : MonoBehaviour
    {
        [System.Serializable] public sealed class Route { public string name; public double distance_m; public List<double[]> points; }

        public string routeResource = "Routes/cugnaux-loop";
        [Tooltip("Metres per second. 1.4 is a walk; stay under 4.2 or the server's speed gate closes.")]
        public float speedMps = 3.0f;
        public bool autoEngage = true;

        public bool Running { get; private set; }
        public double WalkedM { get; private set; }
        public double TotalM { get; private set; }

        private WalkController _controller;
        private MapAdapter _map;
        private List<double[]> _pts;
        private int _seg;
        private double _segDone;
        private float _engageTimer;

        public void Bind(WalkController controller, MapAdapter map) { _controller = controller; _map = map; }

        public void Toggle() { if (Running) Stop(); else Begin(); }

        public void Begin()
        {
            var text = Resources.Load<TextAsset>(routeResource);
            if (text == null) { Debug.LogWarning("WalkSimulator: no route at Resources/" + routeResource); return; }
            var route = JsonConvert.DeserializeObject<Route>(text.text);
            if (route?.points == null || route.points.Count < 2) { Debug.LogWarning("WalkSimulator: route has no points"); return; }
            _pts = route.points;
            TotalM = 0;
            for (int i = 1; i < _pts.Count; i++) TotalM += Geo.DistanceM(_pts[i - 1][0], _pts[i - 1][1], _pts[i][0], _pts[i][1]);
            _seg = 0; _segDone = 0; WalkedM = 0;
            Running = true;
            _map.SetSimulatedLocation(_pts[0][0], _pts[0][1]);
            _controller.SimNote($"walk emulation: {route.name}, {TotalM / 1000:0.0} km at {speedMps:0.0} m/s");
        }

        public void Stop()
        {
            Running = false;
            _controller.SimNote("walk emulation stopped");
        }

        private void Update()
        {
            if (!Running) return;
            double step = speedMps * Time.deltaTime;
            while (step > 0 && _seg < _pts.Count - 1)
            {
                var a = _pts[_seg]; var b = _pts[_seg + 1];
                double len = Geo.DistanceM(a[0], a[1], b[0], b[1]);
                double left = len - _segDone;
                if (step < left) { _segDone += step; WalkedM += step; step = 0; }
                else { step -= left; WalkedM += left; _seg++; _segDone = 0; }
            }
            if (_seg >= _pts.Count - 1)
            {
                _map.SetSimulatedLocation(_pts[_pts.Count - 1][0], _pts[_pts.Count - 1][1]);
                Running = false;
                _controller.SimNote($"walk emulation done: {WalkedM / 1000:0.00} km");
                return;
            }
            var p = _pts[_seg]; var q = _pts[_seg + 1];
            double segLen = Geo.DistanceM(p[0], p[1], q[0], q[1]);
            double t = segLen > 0 ? _segDone / segLen : 0;
            _map.SetSimulatedLocation(p[0] + (q[0] - p[0]) * t, p[1] + (q[1] - p[1]) * t);

            if (!autoEngage) return;
            _engageTimer -= Time.deltaTime;
            if (_engageTimer > 0) return;
            if (_controller.FightActive) { _controller.AutoResolveFight(); _engageTimer = 1.5f; }
            else if (_controller.Modal) { _controller.DismissCard(); _engageTimer = 2f; }
            else { _controller.EngageNearestInRange(); _engageTimer = 1f; }
        }
    }
}
#endif
