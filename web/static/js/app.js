// Native Web Components for Go-Refinery (Light DOM, element-first CSS, SSE integration).

class RefineryDropzone extends HTMLElement {
  connectedCallback() {
    const input = this.querySelector('input[type="file"]');
    const label = this.querySelector("label");

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
      if (input && e.dataTransfer && e.dataTransfer.files.length > 0) {
        input.files = e.dataTransfer.files;
        this.updateFilename(label, input.files[0]);
      }
    });

    if (input) {
      input.addEventListener("change", () => {
        if (input.files && input.files.length > 0) {
          this.updateFilename(label, input.files[0]);
        }
      });
    }
  }

  updateFilename(label, file) {
    if (!label || !file) return;
    const kb = (file.size / 1024).toFixed(1);
    label.textContent = `Selected: ${file.name} (${kb} KB)`;
  }
}

class RefinerySchemaEditor extends HTMLElement {
  connectedCallback() {
    this.syncAllRows();

    this.onChange = (e) => {
      const target = e.target;
      if (target && target.matches('select[data-role="type-selector"]')) {
        const row = target.closest("tr[data-col-row]");
        if (row) {
          this.syncRowRules(row, target.value);
        }
      }
      if (target && target.matches('label[data-rule-chip] input[type="checkbox"]')) {
        const chip = target.closest("label[data-rule-chip]");
        if (chip) {
          chip.setAttribute("data-active", target.checked ? "true" : "false");
        }
      }
    };

    this.addEventListener("change", this.onChange);
  }

  disconnectedCallback() {
    if (this.onChange) {
      this.removeEventListener("change", this.onChange);
    }
  }

  syncAllRows() {
    const rows = this.querySelectorAll("tr[data-col-row]");
    rows.forEach((row) => {
      const select = row.querySelector('select[data-role="type-selector"]');
      if (select) {
        this.syncRowRules(row, select.value);
      }
    });
  }

  syncRowRules(row, selectedType) {
    const chips = row.querySelectorAll("label[data-rule-chip]");
    chips.forEach((chip) => {
      const rawSupported = chip.getAttribute("data-supported-types") || "";
      const supported = rawSupported.split(",").map((s) => s.trim());
      const checkbox = chip.querySelector('input[type="checkbox"]');
      const isCompatible = supported.includes(selectedType);

      if (isCompatible) {
        chip.removeAttribute("hidden");
        if (checkbox) {
          checkbox.disabled = false;
          chip.setAttribute("data-active", checkbox.checked ? "true" : "false");
        }
      } else {
        chip.setAttribute("hidden", "");
        chip.setAttribute("data-active", "false");
        if (checkbox) {
          checkbox.checked = false;
          checkbox.disabled = true;
        }
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

customElements.define("refinery-dropzone", RefineryDropzone);
customElements.define("refinery-schema-editor", RefinerySchemaEditor);
customElements.define("refinery-progress", RefineryProgress);
