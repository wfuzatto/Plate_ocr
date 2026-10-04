# Plate event schema

## nvr.event.plate.detected.v1

Campos do payload específico:

| Campo | Tipo | Obrigatório |
|---|---|---|
| raw_text | string | sim |
| normalized_text | string | sim |
| format | string | sim |
| confidence | number | sim |
| candidates | array | sim |
| detector_confidence | number | sim |
| bbox | array[4] | sim |
| track_id | string | sim |
| lane | string/null | não |
| direction | string/null | não |
| country | string/null | não |

O envelope comum é definido pelo repositório NVR.

## Regras

- `confidence` deve estar entre 0 e 1.
- `bbox` usa coordenadas normalizadas 0..1.
- `raw_text` nunca é reescrito.
- `normalized_text` é resultado do normalizador versionado.
- o evento deve ser idempotente pelo `event_id`.
