# Plate OCR

Plugin oficial de OCR/LPR para a plataforma NVR.

O plugin detecta veículos/placas, executa OCR, agrega leituras entre múltiplos frames e publica eventos normalizados no barramento do NVR. Ele nunca abre RTSP diretamente: os frames pertencem ao Media Engine / Frame Broker do NVR.

## Estado atual — 0.2.0

O núcleo offline já está implementado em Go e não possui dependências externas:

- normalização Mercosul e padrão brasileiro anterior;
- candidatos de ambiguidades OCR sem reescrever silenciosamente a leitura original;
- agregação multi-frame;
- deduplicação temporal por câmera;
- score OCR + detector;
- geração determinística de eventos;
- interfaces desacopladas para detector e OCR;
- pipeline de frame decodificado;
- replay JSONL para testes;
- testes unitários.

O detector de placa e o OCR neural serão conectados atrás das interfaces existentes quando o NVR entregar frames decodificados em pixel format. Modelos ONNX e runtimes de inferência só serão ativados quando estiverem vendorizados no próprio pacote offline.

## Princípios

- Não abre RTSP diretamente.
- Não bloqueia gravação ou live.
- Funciona sem Internet.
- Não baixa modelos em runtime.
- Preserva `raw_text`, candidatos e confidence para auditoria.
- Modelos são substituíveis sem alterar o contrato externo.

## Pipeline

```text
Frame decodificado do NVR
  -> ROI/zone filter
  -> plate detector
  -> crop + perspective correction
  -> OCR top-N
  -> normalization
  -> multi-frame association
  -> confidence aggregation
  -> dedupe
  -> nvr.event.plate.detected.v1
```

## Desenvolvimento offline

```bash
make test
make build
./bin/plate-ocr normalize -text ABC1D23 -confidence 0.94
```

Replay de observações reconhecidas:

```bash
./bin/plate-ocr replay -config configs/default.json < observations.jsonl
```

Veja `docs/ARCHITECTURE.md`, `docs/EVENT_SCHEMA.md` e `docs/IMPLEMENTATION.md`.
