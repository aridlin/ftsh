// Simple "dithering" effect
// Modified: darker background + stronger dithering

const float bayerPattern[4] = float[4](
    0x0514,
    0xC4E6,
    0x3B19,
    0xF7D5
);

float getBayerFromPacked(int x, int y) {
    return float(
        (int(bayerPattern[y & 3]) >> ((x & 3) << 2)) & 0xF
    ) * (1.0 / 16.0);
}

#define LEVELS 36.0              // fewer levels = stronger visible dithering
#define INV_LEVELS (1.0 / LEVELS)

void mainImage(out vec4 fragColor, in vec2 fragCoord)
{
    vec2 uv = fragCoord / iResolution.xy;
    vec3 color = texture(iChannel0, uv).rgb;

    // 🔥 Make background darker
    color *= 0.65;               // 0.85 subtle, 0.65 strong, 0.5 very dark

    // Optional extra contrast
    color = pow(color, vec3(0.9));

    float threshold = getBayerFromPacked(
        int(fragCoord.x),
        int(fragCoord.y)
    );

    // 🔥 Stronger dithering effect
    float ditherStrength = 1.2;  // 1.0 normal, >1 stronger

    vec3 dithered = floor(
        color * LEVELS + threshold * ditherStrength
    ) * INV_LEVELS;

    fragColor = vec4(dithered, 1.0);
}
