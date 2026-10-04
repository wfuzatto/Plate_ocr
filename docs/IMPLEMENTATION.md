# Implementação 0.2.0

Esta etapa transforma a especificação inicial em um núcleo executável e testável sem dependências externas.

## Já implementado

- normalização brasileira Mercosul e padrão antigo;
- preservação de `raw_text`;
- candidatos auditáveis para ambiguidades O/0, I/1, B/8, S/5, G/6 e Z/2;
- agregação multi-frame por câmera, tempo, IoU e similaridade textual;
- score combinado OCR x detector;
- deduplicação temporal por câmera + placa;
- geração determinística de `track_id`, `event_id` e `dedupe_key`;
- métricas atômicas do núcleo;
- interfaces desacopladas `Detector` e `Recognizer`;
- pipeline de frame decodificado -> detector -> OCR -> agregador -> evento;
- replay JSONL offline para testes sem câmera e sem rede;
- testes unitários do normalizador, agregador, dedupe e fluxo completo.

## Integração com o NVR

O NVR atualmente publica Access Units H.264/H.265 no Frame Broker, enquanto o roadmap ainda marca o broker em pixel format como pendente. Esta versão não abre RTSP e não cria uma segunda captura/decoder dentro do plugin.

A integração final receberá NV12/RGB/JPEG do runtime oficial do NVR. Detector e OCR ONNX entram atrás das interfaces de inferência já criadas. Modelos e runtime de inferência deverão estar vendorizados no pacote antes de se tornarem requisito: não haverá download de modelo durante instalação ou execução.

O núcleo gera o evento lógico `nvr.event.plate.detected.v1`. O adaptador do runtime do NVR será responsável por empacotar os atributos específicos no `EventEnvelope` canônico e publicá-los no barramento.

## Replay

Uma observação por linha:

```json
{"camera_id":"cam-1","observed_at":"2026-10-04T18:00:00Z","detector_confidence":0.96,"bbox":{"x1":0.1,"y1":0.2,"x2":0.4,"y2":0.5},"ocr":[{"text":"ABC1D23","confidence":0.94}]}
```

Execução:

```bash
go run ./cmd/plate-ocr replay -config configs/default.json < observations.jsonl
```

A saída é JSONL de eventos consolidados, útil para datasets, testes de regressão e ajuste de thresholds offline.
