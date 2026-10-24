// Native Web Components for Go-Refinery (Light DOM, element-first CSS, SSE integration).

class RefineryDropzone extends HTMLElement {
  connectedCallback() {
    this.addEventListener("dragover", (e) => {
      e.preventDefault();
      this.setAttribute("data-dragover", "true");
    });
    this.addEventListener("dragleave", () => {
      this.removeAttribute("data-dragover");
    });
    this.addEventListener("drop", (e) => {
      e.preventDefault();
      this.removeAttribute("data-dragover");
      const input = this.querySelector('input[type="file"]');
      if (input && e.dataTransfer && e.dataTransfer.files.length > 0) {
        input.files = e.dataTransfer.files;
      }
    });
  }
}

class RefineryProgress extends HTMLElement {
  connectedCallback() {
    const sseUrl = this.getAttribute("sse-url");
    if (!sseUrl) return;

    this.source = new EventSource(sseUrl);
    this.source.addEventListener("progress", (event) => {
      const data = JSON.parse(event.data);
      const bar = this.querySelector("progress");
      const statusOut = this.querySelector('output[data-field="status"]');
      const processedOut = this.querySelector('output[data-field="processed"]');
      const validOut = this.querySelector('output[data-field="valid"]');
      const errorsOut = this.querySelector('output[data-field="errors"]');
      const msgOut = this.querySelector('output[data-field="message"]');

      if (bar) bar.value = data.percentage || 0;
      if (statusOut) {
        statusOut.textContent = data.status;
        statusOut.setAttribute("data-status", data.status);
      }
      if (processedOut) {
        processedOut.textContent = `${data.processed_rows} / ${data.total_rows}`;
      }
      if (validOut) validOut.textContent = String(data.valid_rows);
      if (errorsOut) errorsOut.textContent = String(data.error_rows);
      if (msgOut && data.message) {
        const speed = data.rows_per_second ? ` (${Math.round(data.rows_per_second)} rows/s)` : "";
        msgOut.textContent = `${data.message}${speed}`;
      }

      if (data.status === "COMPLETED" || data.status === "FAILED" || data.status === "CANCELLED") {
        this.source.close();
        this.dispatchEvent(new CustomEvent("refinery:job-finished", { bubbles: true, detail: data }));
        if (window.up && data.status === "COMPLETED") {
          const dsId = this.getAttribute("dataset-id");
          const jobId = this.getAttribute("job-id");
          if (dsId) {
            window.up.navigate({ url: `/datasets/${dsId}/records?job_id=${jobId}`, target: "main" });
          }
        }
      }
    });
  }

  disconnectedCallback() {
    if (this.source) {
      this.source.close();
    }
  }
}

class RefinerySchemaEditor extends HTMLElement {}

customElements.define("refinery-dropzone", RefineryDropzone);
customElements.define("refinery-progress", RefineryProgress);
customElements.define("refinery-schema-editor", RefinerySchemaEditor);
