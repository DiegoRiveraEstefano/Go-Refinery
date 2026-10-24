# Business & Data Workflows (`docs/workflows/index.md`)

This directory documents end-to-end operational flows across multiple hexagonal layers. Use [`workflow-documentation-template.md`](file:///home/diego/Proyectos/Go-Refinery/docs/_templates/workflow-documentation-template.md).

## Core Workflows

1. **Upload & Automatic Schema Inference:**
   `POST /datasets` -> `StorageAdapter.SaveStream` -> `ParserAdapter.ReadSample(1000)` -> `SchemaService.Infer` -> `SQLiteRepository.CreateDataset`.
2. **Asynchronous Refinement & Real-Time Telemetry:**
   `POST /datasets/{id}/jobs` -> `RefineryWorkerPool` -> Stream Rows -> Apply `pipeline.Stage`s -> Batch Insert (`500` rows/tx) -> Broadcast `SSEBroker` -> `<refinery-progress>` updates UI -> Unpoly reloads final results table.
