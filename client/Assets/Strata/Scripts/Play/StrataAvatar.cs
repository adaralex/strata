using UnityEngine;

namespace Strata.Play
{
    /// <summary>
    /// The walker on the map: a human figure built from primitives, no art. Legs and arms
    /// swing from the distance actually covered each frame, the body faces the direction of
    /// travel, and at rest it breathes. The cloak takes a civilization colour so the figure
    /// carries the walker's identity (worn gear, or the ground under them).
    /// </summary>
    public sealed class StrataAvatar : MonoBehaviour
    {
        private static readonly Color Skin = new Color(0.87f, 0.72f, 0.56f);
        private static readonly Color Tunic = new Color(0.30f, 0.22f, 0.16f);
        private static readonly Color Trousers = new Color(0.22f, 0.17f, 0.13f);
        private static readonly Color Belt = new Color(0.85f, 0.68f, 0.35f);
        private static readonly Color Ink = new Color(0.37f, 0.20f, 0.20f);

        private Transform _rig, _hipL, _hipR, _shoulderL, _shoulderR, _torso, _head;
        private Renderer _cloak, _sash;
        private Vector3 _lastPos;
        private float _phase, _heading;
        private bool _hasLast;

        /// <summary>
        /// Put the figure under GO Map's Avatar object, hiding its demo character, or under a
        /// fresh root at the origin when the map has no avatar rig.
        /// </summary>
        public static StrataAvatar Attach()
        {
            var root = GameObject.Find("Avatar");
            if (root == null) root = new GameObject("Avatar");
            foreach (Transform child in root.transform)
                if (child.name == "GoMap Character") child.gameObject.SetActive(false);
            var go = new GameObject("StrataAvatar");
            go.transform.SetParent(root.transform, false);
            var a = go.AddComponent<StrataAvatar>();
            a.Build();
            return a;
        }

        public void SetCloak(Color colour)
        {
            if (_cloak != null) _cloak.material.color = colour;
            if (_sash != null) _sash.material.color = colour;
        }

        private void Build()
        {
            _rig = new GameObject("Rig").transform;
            _rig.SetParent(transform, false);
            // GO Map's demo character stands about ten units tall; the same scale reads from the camera.
            _rig.localScale = Vector3.one * 1.15f;

            var pelvis = Part(_rig, "Pelvis", PrimitiveType.Cube, new Vector3(2.2f, 1.0f, 1.3f), new Vector3(0, 4.4f, 0), Trousers);
            _torso = Part(_rig, "Torso", PrimitiveType.Capsule, new Vector3(2.6f, 2.4f, 1.6f), new Vector3(0, 6.6f, 0), Tunic).transform;
            Part(_rig, "Belt", PrimitiveType.Cube, new Vector3(2.7f, 0.35f, 1.7f), new Vector3(0, 5.1f, 0), Belt);
            _head = Part(_rig, "Head", PrimitiveType.Sphere, new Vector3(1.9f, 2.1f, 1.9f), new Vector3(0, 9.6f, 0), Skin).transform;
            Part(_head, "Hair", PrimitiveType.Sphere, new Vector3(1.05f, 0.7f, 1.05f), new Vector3(0, 0.3f, -0.08f), Ink);
            // Cloak behind, narrower than the shoulders so it reads from the back and the sides;
            // a sash across the chest carries the same colour from the front.
            var cloak = Part(_torso, "Cloak", PrimitiveType.Cube, new Vector3(0.95f, 1.9f, 0.16f), new Vector3(0, -0.45f, -0.62f), Ink);
            _cloak = cloak.GetComponent<Renderer>();
            var sash = Part(_torso, "Sash", PrimitiveType.Cube, new Vector3(0.22f, 1.05f, 0.12f), new Vector3(0.12f, 0.05f, 0.56f), Ink);
            sash.transform.localRotation = Quaternion.Euler(0, 0, 28f);
            _sash = sash.GetComponent<Renderer>();

            _shoulderL = Pivot(_rig, "ShoulderL", new Vector3(-1.55f, 8.0f, 0));
            _shoulderR = Pivot(_rig, "ShoulderR", new Vector3(1.55f, 8.0f, 0));
            Part(_shoulderL, "ArmL", PrimitiveType.Capsule, new Vector3(0.8f, 1.6f, 0.8f), new Vector3(0, -1.5f, 0), Tunic);
            Part(_shoulderR, "ArmR", PrimitiveType.Capsule, new Vector3(0.8f, 1.6f, 0.8f), new Vector3(0, -1.5f, 0), Tunic);
            Part(_shoulderL, "HandL", PrimitiveType.Sphere, new Vector3(0.7f, 0.7f, 0.7f), new Vector3(0, -3.1f, 0), Skin);
            Part(_shoulderR, "HandR", PrimitiveType.Sphere, new Vector3(0.7f, 0.7f, 0.7f), new Vector3(0, -3.1f, 0), Skin);

            _hipL = Pivot(_rig, "HipL", new Vector3(-0.65f, 4.2f, 0));
            _hipR = Pivot(_rig, "HipR", new Vector3(0.65f, 4.2f, 0));
            Part(_hipL, "LegL", PrimitiveType.Capsule, new Vector3(0.95f, 2.0f, 0.95f), new Vector3(0, -1.9f, 0), Trousers);
            Part(_hipR, "LegR", PrimitiveType.Capsule, new Vector3(0.95f, 2.0f, 0.95f), new Vector3(0, -1.9f, 0), Trousers);
            Part(_hipL, "FootL", PrimitiveType.Cube, new Vector3(0.9f, 0.5f, 1.5f), new Vector3(0, -4.0f, 0.3f), Ink);
            Part(_hipR, "FootR", PrimitiveType.Cube, new Vector3(0.9f, 0.5f, 1.5f), new Vector3(0, -4.0f, 0.3f), Ink);

            foreach (var c in GetComponentsInChildren<Collider>(true)) Destroy(c);
        }

        private static Transform Pivot(Transform parent, string name, Vector3 at)
        {
            var p = new GameObject(name).transform;
            p.SetParent(parent, false);
            p.localPosition = at;
            return p;
        }

        private static GameObject Part(Transform parent, string name, PrimitiveType type, Vector3 size, Vector3 at, Color colour)
        {
            var g = GameObject.CreatePrimitive(type);
            g.name = name;
            g.transform.SetParent(parent, false);
            g.transform.localScale = size;
            g.transform.localPosition = at;
            var r = g.GetComponent<Renderer>();
            bool srp = UnityEngine.Rendering.GraphicsSettings.currentRenderPipeline != null;
            var shader = (srp ? Shader.Find("Universal Render Pipeline/Unlit") : null) ?? Shader.Find("Unlit/Color") ?? Shader.Find("Standard");
            r.material = new Material(shader) { color = colour };
            r.shadowCastingMode = UnityEngine.Rendering.ShadowCastingMode.Off;
            r.receiveShadows = false;
            return g;
        }

        private void Update()
        {
            var pos = transform.position;
            if (!_hasLast) { _lastPos = pos; _hasLast = true; return; }
            var delta = pos - _lastPos;
            delta.y = 0;
            _lastPos = pos;
            float dist = delta.magnitude;
            float dt = Mathf.Max(Time.deltaTime, 1e-4f);
            float speed = dist / dt; // map units per second, roughly metres

            if (speed > 0.3f)
            {
                // One stride per ~1.4 m; swing amplitude grows with pace.
                _phase += dist / 1.4f * Mathf.PI * 2f;
                float amp = Mathf.Lerp(20f, 38f, Mathf.Clamp01(speed / 4f));
                float s = Mathf.Sin(_phase);
                _hipL.localRotation = Quaternion.Euler(s * amp, 0, 0);
                _hipR.localRotation = Quaternion.Euler(-s * amp, 0, 0);
                _shoulderL.localRotation = Quaternion.Euler(-s * amp * 0.7f, 0, 0);
                _shoulderR.localRotation = Quaternion.Euler(s * amp * 0.7f, 0, 0);
                _rig.localPosition = new Vector3(0, Mathf.Abs(Mathf.Cos(_phase)) * 0.18f, 0);
                _heading = Mathf.LerpAngle(_heading, Mathf.Atan2(delta.x, delta.z) * Mathf.Rad2Deg, 8f * dt);
            }
            else
            {
                // At rest: limbs settle, a slow breath.
                _hipL.localRotation = Quaternion.Slerp(_hipL.localRotation, Quaternion.identity, 6f * dt);
                _hipR.localRotation = Quaternion.Slerp(_hipR.localRotation, Quaternion.identity, 6f * dt);
                _shoulderL.localRotation = Quaternion.Slerp(_shoulderL.localRotation, Quaternion.Euler(0, 0, 6f), 6f * dt);
                _shoulderR.localRotation = Quaternion.Slerp(_shoulderR.localRotation, Quaternion.Euler(0, 0, -6f), 6f * dt);
                float breath = Mathf.Sin(Time.time * 1.6f);
                _rig.localPosition = new Vector3(0, 0.03f * breath, 0);
                _torso.localScale = new Vector3(2.6f + 0.05f * breath, 2.4f, 1.6f + 0.05f * breath);
            }
            _rig.localRotation = Quaternion.Euler(0, _heading, 0);
        }
    }
}
