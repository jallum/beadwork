# Hypotheses

Six theories about how the prime prompt could be simplified.

## T1: Order matters — actionable state first

If ready/WIP appear before the instructional body, the agent has context about
*what* to do before reading *how*. Might improve task pickup.

## T2: Consistent authority tone outperforms mixed tone

The current prompt shifts from literary ("brilliant and amnesiac") to procedural
to forceful (plan override). If the whole prompt uses a consistently direct
register, the plan mode override might not need as much rhetorical force.

## T3: The opening paragraph is replaceable

"Brilliant and amnesiac" is memorable but may not be load-bearing. A shorter,
direct statement of the problem (context loss) and solution (durable state)
might be equally effective.

## T4: The epic template can be compressed

The inline markdown example is 15 lines. A tighter example plus the mermaid
graph might be sufficient — agents extrapolate from minimal examples.

## T5: Merging sections reduces cognitive overhead

"Durable Notes" (3 lines) could fold into other sections. Fewer sections =
fewer context switches = higher retention.

## T6: The plan mode override can be softened with restructuring

The 5-element override was tested in isolation within a softer prompt. In a
consistently authoritative prompt, fewer elements might suffice.

## T7: Additive > Override (emerged during experiment)

Instead of overriding the agent's plan format, add a materialization step:
"Plan however you want, then create tickets from it before executing." Works
with the agent's instincts instead of against them.
