# Validação e aceitação

O sistema não deve ser declarado adequado para fiscalização ou automação crítica apenas por compilar.

## Dataset mínimo

Separar amostras por dia/noite, chuva/seco, IR ligado/desligado, moto/carro/caminhão, Mercosul/antiga, aproximação/afastamento, velocidade, câmera e lente.

## Métricas

- exact match por passagem;
- false positive por hora/câmera;
- miss rate;
- latência p50/p95/p99;
- frames analisados/s;
- eventos duplicados;
- eventos perdidos durante indisponibilidade do NVR.

## Gate de implantação

O provider clássico é o baseline. Para cada ponto, medir se a precisão atende ao objetivo. Se não atender, substituir somente o provider de inferência por um modelo neural validado, mantendo arquitetura e contratos.
