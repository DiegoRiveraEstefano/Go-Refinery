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
    this.draggedBlock = null;
    this.syncAllPanels();

    this.onClick = (e) => {
      const tabBtn = e.target.closest('button[data-role="col-tab"]');
      if (tabBtn) {
        this.selectColumn(tabBtn.getAttribute("data-col-index"));
        return;
      }

      const addBtn = e.target.closest('button[data-role="add-stage-btn"]');
      if (addBtn) {
        const panel = addBtn.closest("fieldset[data-col-panel]");
        if (panel) {
          this.addSelectedStage(panel);
        }
        return;
      }

      const moveUpBtn = e.target.closest('button[data-role="move-block-up"]');
      if (moveUpBtn) {
        const block = moveUpBtn.closest('li[data-block-kind="stage"]');
        const panel = moveUpBtn.closest("fieldset[data-col-panel]");
        if (block && panel) {
          this.moveBlock(panel, block, -1);
        }
        return;
      }

      const moveDownBtn = e.target.closest('button[data-role="move-block-down"]');
      if (moveDownBtn) {
        const block = moveDownBtn.closest('li[data-block-kind="stage"]');
        const panel = moveDownBtn.closest("fieldset[data-col-panel]");
        if (block && panel) {
          this.moveBlock(panel, block, 1);
        }
        return;
      }

      const removeBtn = e.target.closest('button[data-role="remove-block"]');
      if (removeBtn) {
        const block = removeBtn.closest('li[data-block-kind="stage"]');
        const panel = removeBtn.closest("fieldset[data-col-panel]");
        if (block && panel) {
          block.remove();
          this.syncPanel(panel);
        }
      }
    };

    this.onChange = (e) => {
      const target = e.target;
      if (!target) return;

      if (target.matches('select[data-role="type-selector"]')) {
        const panel = target.closest("fieldset[data-col-panel]");
        if (panel) {
          this.syncPanel(panel);
        }
      }
    };

    this.onInput = (e) => {
      const target = e.target;
      if (target && target.matches('input[data-role="key-input"]')) {
        const panel = target.closest("fieldset[data-col-panel]");
        if (panel) {
          this.syncSidebarTab(panel);
        }
      }
    };

    this.onDragStart = (e) => {
      const block = e.target.closest('li[data-block-kind="stage"]');
      if (!block) return;
      this.draggedBlock = block;
      block.setAttribute("data-dragging", "true");
      if (e.dataTransfer) {
        e.dataTransfer.effectAllowed = "move";
        e.dataTransfer.setData("text/plain", block.getAttribute("data-rule-type") || "");
      }
    };

    this.onDragOver = (e) => {
      if (!this.draggedBlock) return;
      const targetBlock = e.target.closest('li[data-block-kind="stage"]');
      if (!targetBlock || targetBlock === this.draggedBlock) return;

      const sourceStack = this.draggedBlock.parentElement;
      if (targetBlock.parentElement !== sourceStack) return;

      e.preventDefault();
      if (e.dataTransfer) {
        e.dataTransfer.dropEffect = "move";
      }

      const rect = targetBlock.getBoundingClientRect();
      const offset = e.clientY - rect.top;
      if (offset < rect.height / 2) {
        sourceStack.insertBefore(this.draggedBlock, targetBlock);
      } else {
        sourceStack.insertBefore(this.draggedBlock, targetBlock.nextSibling);
      }
    };

    this.onDragEnd = () => {
      if (!this.draggedBlock) return;
      const panel = this.draggedBlock.closest("fieldset[data-col-panel]");
      this.draggedBlock.removeAttribute("data-dragging");
      this.draggedBlock = null;
      if (panel) {
        this.syncPanel(panel);
      }
    };

    this.addEventListener("click", this.onClick);
    this.addEventListener("change", this.onChange);
    this.addEventListener("input", this.onInput);
    this.addEventListener("dragstart", this.onDragStart);
    this.addEventListener("dragover", this.onDragOver);
    this.addEventListener("dragend", this.onDragEnd);
  }

  disconnectedCallback() {
    if (this.onClick) this.removeEventListener("click", this.onClick);
    if (this.onChange) this.removeEventListener("change", this.onChange);
    if (this.onInput) this.removeEventListener("input", this.onInput);
    if (this.onDragStart) this.removeEventListener("dragstart", this.onDragStart);
    if (this.onDragOver) this.removeEventListener("dragover", this.onDragOver);
    if (this.onDragEnd) this.removeEventListener("dragend", this.onDragEnd);
  }

  selectColumn(colIndex) {
    if (colIndex === null || colIndex === undefined) return;
    const activeInput = this.querySelector('input[data-role="active-col-input"]');
    if (activeInput) {
      activeInput.value = String(colIndex);
    }

    const tabs = this.querySelectorAll('button[data-role="col-tab"]');
    tabs.forEach((tab) => {
      const isSelected = tab.getAttribute("data-col-index") === String(colIndex);
      tab.setAttribute("aria-selected", isSelected ? "true" : "false");
    });

    const panels = this.querySelectorAll("fieldset[data-col-panel]");
    panels.forEach((panel) => {
      const isMatch = panel.getAttribute("data-col-panel") === String(colIndex);
      if (isMatch) {
        panel.removeAttribute("hidden");
      } else {
        panel.setAttribute("hidden", "");
      }
    });
  }

  syncAllPanels() {
    const panels = this.querySelectorAll("fieldset[data-col-panel]");
    panels.forEach((panel) => this.syncPanel(panel));
  }

  syncPanel(panel) {
    const select = panel.querySelector('select[data-role="type-selector"]');
    const selectedType = select ? select.value : "STRING";
    const stack = panel.querySelector('ol[data-role="block-stack"]');
    const stageBlocks = stack ? Array.from(stack.querySelectorAll('li[data-block-kind="stage"]')) : [];
    const activeRuleTypes = new Set();

    stageBlocks.forEach((block, idx) => {
      const ruleType = block.getAttribute("data-rule-type") || "";
      activeRuleTypes.add(ruleType);

      const stepBadge = block.querySelector('[data-role="step-badge"]');
      if (stepBadge) {
        stepBadge.textContent = `Block ${idx + 2}`;
      }

      const upBtn = block.querySelector('button[data-role="move-block-up"]');
      const downBtn = block.querySelector('button[data-role="move-block-down"]');
      if (upBtn) upBtn.disabled = idx === 0;
      if (downBtn) downBtn.disabled = idx === stageBlocks.length - 1;

      const rawSupported = block.getAttribute("data-supported-types") || "";
      const supported = rawSupported
        .split(",")
        .map((s) => s.trim())
        .filter(Boolean);
      const isCompatible = supported.includes(selectedType);

      block.setAttribute("data-compatible", isCompatible ? "true" : "false");
      const hiddenInput = block.querySelector('input[type="hidden"]');
      if (hiddenInput) {
        hiddenInput.disabled = !isCompatible;
      }

      const notice = block.querySelector('[data-role="incompatible-notice"]');
      if (notice) {
        if (isCompatible) {
          notice.setAttribute("hidden", "");
        } else {
          notice.removeAttribute("hidden");
        }
      }
    });

    const picker = panel.querySelector('select[data-role="stage-picker"]');
    if (picker) {
      const options = picker.querySelectorAll("option[ value ]");
      options.forEach((opt) => {
        const val = opt.getAttribute("value");
        if (!val) return;
        const rawSupported = opt.getAttribute("data-supported-types") || "";
        const supported = rawSupported
          .split(",")
          .map((s) => s.trim())
          .filter(Boolean);
        const isCompatible = supported.includes(selectedType);
        const isAlreadyStacked = activeRuleTypes.has(val);
        const baseLabel = `[${opt.getAttribute("data-category")}] ${opt.getAttribute("data-label")} — ${opt.getAttribute("data-description")}`;

        if (!isCompatible) {
          opt.setAttribute("hidden", "");
          opt.disabled = true;
          opt.textContent = baseLabel;
        } else {
          opt.removeAttribute("hidden");
          opt.disabled = isAlreadyStacked;
          opt.textContent = isAlreadyStacked ? `${baseLabel} (Stacked)` : baseLabel;
        }
      });

      if (picker.selectedOptions.length > 0 && picker.selectedOptions[0].disabled) {
        picker.value = "";
      }
    }

    this.syncSidebarTab(panel);
  }

  syncSidebarTab(panel) {
    const colIndex = panel.getAttribute("data-col-panel");
    const tab = this.querySelector(`button[data-role="col-tab"][data-col-index="${colIndex}"]`);
    if (!tab) return;

    const select = panel.querySelector('select[data-role="type-selector"]');
    const keyInput = panel.querySelector('input[data-role="key-input"]');
    const stageBlocks = panel.querySelectorAll('li[data-block-kind="stage"]');
    const incompatibleBlocks = panel.querySelectorAll('li[data-block-kind="stage"][data-compatible="false"]');

    const typeBadge = tab.querySelector('[data-role="col-tab-type"]');
    if (typeBadge && select) {
      typeBadge.textContent = select.value;
    }

    const keyLabel = tab.querySelector('[data-role="col-tab-key"]');
    if (keyLabel && keyInput) {
      keyLabel.textContent = keyInput.value;
    }

    const countBadge = tab.querySelector('[data-role="col-tab-count"]');
    if (countBadge) {
      countBadge.textContent = `+${stageBlocks.length}`;
    }

    const warnBadge = tab.querySelector('[data-role="col-tab-warn"]');
    if (warnBadge) {
      if (incompatibleBlocks.length > 0) {
        warnBadge.removeAttribute("hidden");
      } else {
        warnBadge.setAttribute("hidden", "");
      }
    }
  }

  addSelectedStage(panel) {
    const picker = panel.querySelector('select[data-role="stage-picker"]');
    const stack = panel.querySelector('ol[data-role="block-stack"]');
    if (!picker || !stack || !picker.value) {
      if (picker) picker.focus();
      return;
    }

    const opt = picker.selectedOptions[0];
    if (!opt || opt.disabled) return;

    const colIndex = panel.getAttribute("data-col-panel");
    const ruleType = opt.value;
    const label = opt.getAttribute("data-label") || ruleType;
    const description = opt.getAttribute("data-description") || "";
    const category = opt.getAttribute("data-category") || "COMMON";
    const supportedCSV = opt.getAttribute("data-supported-types") || "";

    const li = document.createElement("li");
    li.setAttribute("data-block-card", "");
    li.setAttribute("data-block-kind", "stage");
    li.setAttribute("data-rule-type", ruleType);
    li.setAttribute("data-category", category);
    li.setAttribute("data-supported-types", supportedCSV);
    li.setAttribute("data-compatible", "true");
    li.setAttribute("draggable", "true");

    const hiddenInput = document.createElement("input");
    hiddenInput.type = "hidden";
    hiddenInput.name = `col_${colIndex}_rules`;
    hiddenInput.value = ruleType;

    const topRow = document.createElement("div");
    topRow.className = "l-cluster";

    const titleGroup = document.createElement("div");
    titleGroup.className = "l-cluster";

    const dragHandle = document.createElement("span");
    dragHandle.setAttribute("data-role", "drag-handle");
    dragHandle.setAttribute("aria-hidden", "true");
    dragHandle.setAttribute("title", "Drag block to reorder");
    dragHandle.textContent = "\u22EE\u22EE";

    const stepBadge = document.createElement("span");
    stepBadge.className = "u-badge u-mono";
    stepBadge.setAttribute("data-role", "step-badge");
    stepBadge.textContent = "Block";

    const strong = document.createElement("strong");
    strong.textContent = label;

    const catBadge = document.createElement("span");
    catBadge.className = "u-badge u-mono";
    catBadge.setAttribute("data-category-badge", category);
    catBadge.textContent = category;

    titleGroup.append(dragHandle, stepBadge, strong, catBadge);

    const actionsGroup = document.createElement("div");
    actionsGroup.className = "l-cluster";
    actionsGroup.setAttribute("data-role", "block-actions");

    const upBtn = document.createElement("button");
    upBtn.type = "button";
    upBtn.setAttribute("data-variant", "secondary");
    upBtn.setAttribute("data-role", "move-block-up");
    upBtn.setAttribute("aria-label", `Move ${label} up`);
    upBtn.setAttribute("title", "Move block up");
    upBtn.textContent = "\u2191";

    const downBtn = document.createElement("button");
    downBtn.type = "button";
    downBtn.setAttribute("data-variant", "secondary");
    downBtn.setAttribute("data-role", "move-block-down");
    downBtn.setAttribute("aria-label", `Move ${label} down`);
    downBtn.setAttribute("title", "Move block down");
    downBtn.textContent = "\u2193";

    const removeBtn = document.createElement("button");
    removeBtn.type = "button";
    removeBtn.setAttribute("data-variant", "secondary");
    removeBtn.setAttribute("data-role", "remove-block");
    removeBtn.setAttribute("aria-label", `Remove ${label}`);
    removeBtn.setAttribute("title", "Remove block");
    removeBtn.textContent = "\u00D7";

    actionsGroup.append(upBtn, downBtn, removeBtn);
    topRow.append(titleGroup, actionsGroup);

    const descP = document.createElement("p");
    descP.textContent = description;

    const notice = document.createElement("small");
    notice.setAttribute("data-role", "incompatible-notice");
    notice.setAttribute("hidden", "");
    notice.textContent = `Incompatible with selected Target Data Type — preserved in stack but excluded from execution until removed or type is restored (supports: ${supportedCSV}).`;

    li.append(hiddenInput, topRow, descP, notice);
    stack.appendChild(li);

    picker.value = "";
    this.syncPanel(panel);
  }

  moveBlock(panel, block, direction) {
    const stack = panel.querySelector('ol[data-role="block-stack"]');
    if (!stack) return;
    const stages = Array.from(stack.querySelectorAll('li[data-block-kind="stage"]'));
    const currentIndex = stages.indexOf(block);
    if (currentIndex === -1) return;

    const targetIndex = currentIndex + direction;
    if (targetIndex < 0 || targetIndex >= stages.length) return;

    const referenceBlock = stages[targetIndex];
    if (direction < 0) {
      stack.insertBefore(block, referenceBlock);
    } else {
      stack.insertBefore(block, referenceBlock.nextSibling);
    }

    this.syncPanel(panel);
  }
}

class RefinerySearchBox extends HTMLElement {
  connectedCallback() {
    const appendBtn = this.querySelector('button[data-role="append-mongo"]');
    const queryInput = this.querySelector('input[data-role="query-input"]');
    const fieldSelect = this.querySelector('select[data-role="builder-field"]');
    const opSelect = this.querySelector('select[data-role="builder-op"]');
    const valInput = this.querySelector('input[data-role="builder-val"]');

    if (appendBtn && queryInput && fieldSelect && opSelect && valInput) {
      this.onAppendClick = () => {
        const field = fieldSelect.value === "*" ? "" : fieldSelect.value;
        const op = opSelect.value || "$contains";
        const rawVal = valInput.value.trim();
        if (!field) {
          fieldSelect.focus();
          return;
        }
        if (!rawVal && op !== "$exists") {
          valInput.focus();
          return;
        }

        let currentDoc = {};
        const existing = queryInput.value.trim();
        if (existing.startsWith("{") && existing.endsWith("}")) {
          try {
            currentDoc = JSON.parse(existing);
          } catch (_) {
            currentDoc = {};
          }
        }

        let parsedVal = rawVal;
        if (op === "$in") {
          parsedVal = rawVal
            .split(",")
            .map((s) => s.trim())
            .filter(Boolean);
        } else if (op === "$exists") {
          parsedVal = rawVal.toLowerCase() !== "false" && rawVal !== "0";
        } else if (rawVal !== "" && !Number.isNaN(Number(rawVal)) && op !== "$regex" && op !== "$contains") {
          parsedVal = Number(rawVal);
        }

        if (
          typeof currentDoc[field] === "object" &&
          currentDoc[field] !== null &&
          !Array.isArray(currentDoc[field])
        ) {
          currentDoc[field][op] = parsedVal;
        } else {
          currentDoc[field] = { [op]: parsedVal };
        }

        queryInput.value = JSON.stringify(currentDoc);
        valInput.value = "";
      };

      appendBtn.addEventListener("click", this.onAppendClick);
    }
  }

  disconnectedCallback() {
    const appendBtn = this.querySelector('button[data-role="append-mongo"]');
    if (appendBtn && this.onAppendClick) {
      appendBtn.removeEventListener("click", this.onAppendClick);
    }
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
customElements.define("refinery-search-box", RefinerySearchBox);
customElements.define("refinery-progress", RefineryProgress);
