#!/usr/bin/env python3
"""Generate lather hero visual (GitHub dark palette, no-cringe flat design)."""
from PIL import Image, ImageDraw, ImageFont

BG = (13, 17, 23)          # #0d1117
FG = (230, 237, 243)       # #e6edf3
MUTED = (139, 148, 158)    # #8b949e
GO = (0, 173, 216)         # Go cyan #00ADD8
SOAP = (255, 200, 87)      # warm soap-bar yellow
BORDER = (48, 54, 61)      # #30363d

SANS = "/usr/share/fonts/truetype/noto/NotoSans-Bold.ttf"
SANS_R = "/usr/share/fonts/truetype/noto/NotoSans-Regular.ttf"
MONO = "/usr/share/fonts/truetype/dejavu/DejaVuSansMono.ttf"


def font(path, size):
    return ImageFont.truetype(path, size)


def draw_bubbles(d, cx, cy, s, color):
    """Flat soap bar with three bubbles, geometric, no gradients."""
    lw = max(8, int(s * 0.06))
    # rounded soap bar
    d.rounded_rectangle([cx - s * 0.55, cy - s * 0.18, cx + s * 0.55, cy + s * 0.42],
                        radius=int(s * 0.16), outline=color, width=lw)
    # bubbles rising from the bar
    for i, (bx, by, br) in enumerate([(-0.30, -0.42, 0.13), (0.02, -0.62, 0.10), (0.30, -0.38, 0.08)]):
        d.ellipse([cx + s * bx - s * br, cy + s * by - s * br,
                   cx + s * bx + s * br, cy + s * by + s * br], outline=color, width=lw)
    # shine line on the bar
    d.line([cx - s * 0.34, cy - s * 0.02, cx + s * 0.05, cy - s * 0.02], fill=color, width=lw)


def hero():
    W, H = 1280, 640
    img = Image.new("RGB", (W, H), BG)
    d = ImageDraw.Draw(img)
    for x in range(0, W, 64):
        d.line([x, 0, x, H], fill=(20, 25, 32))
    for y in range(0, H, 64):
        d.line([0, y, W, y], fill=(20, 25, 32))
    draw_bubbles(d, 1100, 225, 290, SOAP)
    d.text((90, 210), "lather", font=font(SANS, 120), fill=FG)
    d.text((94, 370), "WSDL/SOAP toolkit for Go. Collections included.", font=font(SANS_R, 44), fill=MUTED)
    d.text((94, 500), "$ go install github.com/ofershap/lather/cmd/lather@latest",
           font=font(MONO, 28), fill=GO)
    img.save("assets/hero.png")


hero()
