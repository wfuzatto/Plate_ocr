# Plate OCR — arquitetura

## 1. Responsabilidade

O plugin transforma frames fornecidos pelo NVR em eventos confiáveis de passagem de placas.

Ele não é responsável por:
- RTSP;
- autenticação da câmera;
- retenção de vídeo;
- live view;
- usuários;
- armazenamento de gravação.

## 2. Pipeline detalhado

### 2.1 Frame admission
Cada câmera possui configuração:
- ativo/inativo;
- ROI;
- sentido/faixa;
- FPS de análise;
- resolução mínima;
- threshold;
- cooldown/dedupe.

Frames fora da ROI podem ser descartados cedo.

### 2.2 Detecção
Detector localiza placa e retorna bounding box + confidence.

A interface do detector deve ser independente do modelo:
```
Detector.detect(frame) -> PlateBox[]
```

### 2.3 Crop/retificação
- crop com margem;
- correção de perspectiva opcional;
- normalização de iluminação;
- upscale opcional;
- preservação do crop original para auditoria.

### 2.4 OCR
OCR retorna top-N candidatos, não apenas uma string:
```
[
  { text, confidence },
  ...
]
```

### 2.5 Normalização
A normalização:
- uppercase;
- remove caracteres inválidos;
- classifica padrão provável;
- pontua candidatos compatíveis;
- nunca apaga `raw_text`.

### 2.6 Tracking temporal
Uma passagem pode gerar leituras em vários frames.

O agregador relaciona detecções usando:
- tempo;
- posição/bbox;
- camera;
- similaridade textual;
- opcionalmente vehicle tracker.

Só depois consolida a leitura final.

### 2.7 Deduplicação
Evita publicar dezenas de eventos para o mesmo veículo.

A dedupe key pode considerar:
```
camera_id + normalized_plate + time_bucket
```

A janela é configurável por câmera.

## 3. Evento

Exemplo lógico:

```json
{
  "event_type": "nvr.event.plate.detected.v1",
  "camera_id": "...",
  "observed_at": "...",
  "plate": {
    "raw_text": "ABC1D23",
    "normalized_text": "ABC1D23",
    "format": "BR_MERCOSUL",
    "confidence": 0.94,
    "candidates": [
      {"text": "ABC1D23", "confidence": 0.94}
    ]
  },
  "detector": {
    "confidence": 0.97,
    "bbox": [0.1, 0.4, 0.25, 0.52]
  },
  "snapshot_ref": "...",
  "track_id": "..."
}
```

## 4. Pesquisa e hotlist

O plugin publica eventos; a pesquisa pertence ao NVR.

O core indexará:
- placa exata;
- parcial;
- horário;
- câmera;
- cidade/site;
- confidence;
- direção/faixa;
- listas de interesse.

Hotlists devem gerar evento separado e auditável.

## 5. CPU/GPU

Implementar provider abstraction:

```
InferenceProvider
  ├── CPU
  ├── CUDA
  ├── TensorRT
  └── future providers
```

O container GPU não deve ser obrigatório para desenvolvimento.

## 6. Performance

Métricas:
- received_fps;
- analyzed_fps;
- dropped_frames;
- detector_latency_ms;
- ocr_latency_ms;
- end_to_end_latency_ms;
- detections_total;
- accepted_reads_total;
- rejected_reads_total;
- confidence histogram;
- queue_depth.

Quando houver overload:
1. não bloquear frame broker;
2. descartar frames intermediários;
3. priorizar frames mais recentes;
4. reportar DEGRADED.

## 7. Qualidade

O benchmark deve ser separado por:
- dia/noite;
- chuva;
- ângulo;
- velocidade;
- distância;
- Mercosul/antiga;
- IR/reflexo;
- moto/carro/caminhão.

Métrica principal não será apenas OCR character accuracy, mas:
- plate exact-match rate;
- false positive rate;
- missed plate rate;
- latency;
- leitura por passagem.

## 8. Segurança

O plugin recebe somente:
- frames autorizados;
- configuração necessária;
- identidade da câmera;
- token curto do runtime.

Não recebe credencial RTSP.

## 9. Testes

Criar datasets versionados por manifesto, sem commitar material sensível no repositório.

Testes:
- unitários de normalização;
- golden images;
- replay de vídeo;
- benchmark CPU/GPU;
- restart durante carga;
- fila saturada;
- frames corrompidos;
- baixa iluminação.
