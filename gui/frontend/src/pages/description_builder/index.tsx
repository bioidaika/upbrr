// Copyright (c) 2025-2026, Audionut and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

import type { Dispatch, SetStateAction } from "react";
import type { DescriptionBuilderPreview } from "../../types";
import { handleExternalLinkClick } from "../../utils/externalLinks";

type Props = {
  path: string;
  builderPreview: DescriptionBuilderPreview;
  builderRaw: string;
  builderRenderedHTML: string;
  builderLoading: boolean;
  builderSaving: boolean;
  builderRenderLoading: boolean;
  builderError: string;
  builderSaved: string;
  setBuilderRaw: Dispatch<SetStateAction<string>>;
  setBuilderDirty: Dispatch<SetStateAction<boolean>>;
  resetBuilderDescription: () => void;
  renderBuilderDescription: () => void;
  saveBuilderDescription: () => void;
};

const decodeHtmlEntities = (value: string) => {
  if (!value) return value;
  if (!value.includes("&lt;") && !value.includes("&gt;") && !value.includes("&#")) {
    return value;
  }
  const textarea = document.createElement("textarea");
  textarea.innerHTML = value;
  return textarea.value;
};

const renderImageHostSummary = (message: string, trackers: string[], reuploaded: boolean) => {
  if (!message || !reuploaded) return null;
  const label = trackers.length > 0 ? trackers.join(", ") : "Trackers";
  return (
    <div className="builder-host-status" key={`${label}-${message}`}>
      <p className="label">{label}</p>
      <p className="muted">{message}</p>
    </div>
  );
};

export default function DescriptionBuilderPage(props: Props) {
  const {
    path,
    builderPreview,
    builderRaw,
    builderRenderedHTML,
    builderLoading,
    builderSaving,
    builderRenderLoading,
    builderError,
    builderSaved,
    setBuilderRaw,
    setBuilderDirty,
    resetBuilderDescription,
    renderBuilderDescription,
    saveBuilderDescription
  } = props;
  const hasImageHostUploads = (builderPreview.ImageHosts || []).some(
    (entry) => Boolean(entry.ImageHost?.Reuploaded) && Boolean(entry.ImageHost?.Message)
  );

  return (
    <section className="builder-panel">
      <header className="builder-header">
        <p className="eyebrow">Description Builder</p>
        <h1>Customize Description</h1>
        <p className="subtitle">
          Review and edit the base description before tracker-specific formatting.
        </p>
      </header>

      <section className="panel builder-actions">
        <div>
          <p className="label">Source path</p>
          <p className="value dupe-path">{path || "No path selected"}</p>
          {builderPreview.HasOverride ? (
            <p className="muted">Saved override is active for this path.</p>
          ) : null}
        </div>
        <div className="builder-actions__buttons">
          <button
            className="ghost"
            type="button"
            onClick={resetBuilderDescription}
            disabled={builderLoading || builderSaving || !path.trim()}
          >
            {builderLoading ? "Resetting..." : "Reset description"}
          </button>
          <button
            className="ghost"
            type="button"
            onClick={renderBuilderDescription}
            disabled={builderRenderLoading || !builderRaw.trim()}
          >
            {builderRenderLoading ? "Rendering..." : "Render"}
          </button>
          <button
            className="primary"
            type="button"
            onClick={saveBuilderDescription}
            disabled={builderSaving || !path.trim()}
          >
            {builderSaving ? "Saving..." : "Save and continue"}
          </button>
        </div>
      </section>

      {builderError ? <p className="error">{builderError}</p> : null}
      {builderSaved ? <p className="success">{builderSaved}</p> : null}

      {hasImageHostUploads ? (
        <section className="panel builder-actions">
          <div>
            <p className="label">Image host status</p>
            <div className="builder-host-statuses">
              {builderPreview.ImageHosts.map((entry) =>
                renderImageHostSummary(entry.ImageHost?.Message || "", entry.Trackers || [], Boolean(entry.ImageHost?.Reuploaded))
              )}
            </div>
          </div>
        </section>
      ) : null}

      <section className="panel builder-editor">
        <div className="builder-editor__header">
          <h2>Raw Description</h2>
          <p className="muted">Edit BBCode or HTML directly. Save to apply across trackers.</p>
        </div>
        <textarea
          className="builder-textarea"
          value={builderRaw}
          onChange={(event) => {
            setBuilderRaw(event.target.value);
            setBuilderDirty(true);
          }}
          placeholder="Reset the description first, then edit it here."
        />
      </section>

      <section className="panel builder-preview">
        <div className="builder-preview__header">
          <h2>Rendered Preview</h2>
        </div>
        {builderRenderedHTML ? (
          <div
            className="tracker-description rendered"
            onClick={handleExternalLinkClick}
            dangerouslySetInnerHTML={{ __html: decodeHtmlEntities(builderRenderedHTML) }}
          />
        ) : (
          <p className="muted">No rendered preview yet.</p>
        )}
      </section>
    </section>
  );
}
