# Plate OCR

Plugin oficial de OCR/LPR para a plataforma NVR.

O plugin detecta veículos/placas, executa OCR, agrega leituras entre múltiplos frames e publica eventos normalizados no barramento do NVR.

## Princípios

- Não abre RTSP diretamente.
- Recebe frames do Frame Broker do NVR.
- Pode executar em CPU ou GPU.
- Não bloqueia gravação/live.
- É idempotente.
- Faz deduplicação temporal por câmera.
- Mantém candidates/confidence para auditoria da leitura.
- Permite modelos substituíveis sem alterar o contrato externo.

## Pipeline

```
Frame
  -> ROI/zone filter
  -> plate detector
  -> crop + perspective correction
  -> OCR
  -> normalization
  -> multi-frame association
  -> confidence aggregation
  -> dedupe
  -> plate.detected event
```

## Runtime de inferência

A camada de inferência será abstrata. Baseline:
- ONNX Runtime;
- CPU provider como fallback;
- CUDA/TensorRT quando disponível;
- possibilidade futura de OpenVINO/AMD sem mudar a API do plugin.

Modelos específicos não serão acoplados ao core.

## Placas brasileiras

O normalizador deve reconhecer e pontuar:
- padrão Mercosul;
- padrão brasileiro anterior;
- confusões OCR comuns (O/0, I/1, B/8 etc.) apenas como candidatos, sem alterar silenciosamente a leitura original.

Sempre armazenar:
- `raw_text`;
- `normalized_text`;
- `candidates`;
- `ocr_confidence`;
- `detector_confidence`;
- evidência visual quando autorizada.

## Evento principal

`nvr.event.plate.detected.v1`

A especificação detalhada está em `docs/ARCHITECTURE.md`.

## Status

Fase 0 — arquitetura/contrato.
