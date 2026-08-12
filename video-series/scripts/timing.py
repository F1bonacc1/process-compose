"""Beat timing shared by the video renderer and the script exporter.

A beat is one slide plus its narration clip. The slide appears slightly before
the voice starts and stays up slightly after it stops, so the two never change
at the same instant. Both numbers live here so SCRIPTS.md timestamps and the
rendered video agree.
"""

LEAD_S = 0.35   # slide is on screen this long before the narration starts
TAIL_S = 0.55   # silence after the narration before the next slide
GAP_S = LEAD_S + TAIL_S
