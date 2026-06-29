import type { ChangeEvent, ReactNode } from 'react';
import { useEffect, useRef, useState } from 'react';
import AccountTreeRoundedIcon from '@mui/icons-material/AccountTreeRounded';
import AddRoundedIcon from '@mui/icons-material/AddRounded';
import ContentCopyRoundedIcon from '@mui/icons-material/ContentCopyRounded';
import DeleteOutlineRoundedIcon from '@mui/icons-material/DeleteOutlineRounded';
import FileDownloadRoundedIcon from '@mui/icons-material/FileDownloadRounded';
import FileUploadRoundedIcon from '@mui/icons-material/FileUploadRounded';
import FiberManualRecordRoundedIcon from '@mui/icons-material/FiberManualRecordRounded';
import KeyboardRoundedIcon from '@mui/icons-material/KeyboardRounded';
import MouseRoundedIcon from '@mui/icons-material/MouseRounded';
import OpenWithRoundedIcon from '@mui/icons-material/OpenWithRounded';
import SwapVertRoundedIcon from '@mui/icons-material/SwapVertRounded';
import TextFieldsRoundedIcon from '@mui/icons-material/TextFieldsRounded';
import PlayArrowRoundedIcon from '@mui/icons-material/PlayArrowRounded';
import SaveRoundedIcon from '@mui/icons-material/SaveRounded';
import StopRoundedIcon from '@mui/icons-material/StopRounded';
import TimerRoundedIcon from '@mui/icons-material/TimerRounded';
import ArrowUpwardRoundedIcon from '@mui/icons-material/ArrowUpwardRounded';
import ArrowDownwardRoundedIcon from '@mui/icons-material/ArrowDownwardRounded';
import { Box, Switch, Typography, useTheme } from '@mui/material';
import type { SxProps, Theme } from '@mui/material';
import { useT } from '@/i18n';
import { NativeDialogs, type MacroRepeatMode, type MacroStep, type MacroStepKind, type MacroSummary } from '@/services';
import { macroPageStyles } from './MacroPage.styles';
import { useMacroPage, type UseMacroPageResult } from './useMacroPage';

export const MacroPage = () => {
  const theme = useTheme();
  const styles = macroPageStyles(theme);
  const vm = useMacroPage();
  const t = useT();

  const selectedStep = vm.draft?.steps[vm.selectedStepIndex] ?? null;
  const running = vm.runningState === 'running' || vm.runningState === 'stopping';
  const [dragIndex, setDragIndex] = useState<number | null>(null);
  const [dragOverIndex, setDragOverIndex] = useState<number | null>(null);
  const [selectedIndices, setSelectedIndices] = useState<Set<number>>(new Set());
  const rangeDragStartRef = useRef<number | null>(null);
  const rangeHasDraggedRef = useRef(false);

  // Clear multi-selection when switching macros.
  useEffect(() => { setSelectedIndices(new Set()); }, [vm.activeId]);
  // Native HTML5 drag suppresses wheel events in WebView2, so the block list
  // cannot be scrolled with the mouse wheel while dragging. We emulate the
  // common "edge auto-scroll" behaviour: while a drag hovers near the top or
  // bottom edge of the rail, a rAF loop scrolls the rail in that direction.
  const railRef = useRef<HTMLDivElement | null>(null);
  const autoScrollRef = useRef<{ raf: number; velocity: number }>({ raf: 0, velocity: 0 });

  const stopAutoScroll = (): void => {
    if (autoScrollRef.current.raf) {
      cancelAnimationFrame(autoScrollRef.current.raf);
      autoScrollRef.current.raf = 0;
    }
    autoScrollRef.current.velocity = 0;
  };

  const runAutoScroll = (): void => {
    const rail = railRef.current;
    if (!rail || autoScrollRef.current.velocity === 0) {
      autoScrollRef.current.raf = 0;
      return;
    }
    rail.scrollTop += autoScrollRef.current.velocity;
    autoScrollRef.current.raf = requestAnimationFrame(runAutoScroll);
  };

  const updateAutoScroll = (clientY: number): void => {
    const rail = railRef.current;
    if (!rail) return;
    const rect = rail.getBoundingClientRect();
    const edge = 48; // px from each edge that triggers auto-scroll
    const maxSpeed = 14; // px per frame at the very edge
    let velocity = 0;
    if (clientY < rect.top + edge) {
      const ratio = Math.min(1, (rect.top + edge - clientY) / edge);
      velocity = -Math.ceil(ratio * maxSpeed);
    } else if (clientY > rect.bottom - edge) {
      const ratio = Math.min(1, (clientY - (rect.bottom - edge)) / edge);
      velocity = Math.ceil(ratio * maxSpeed);
    }
    autoScrollRef.current.velocity = velocity;
    if (velocity !== 0 && !autoScrollRef.current.raf) {
      autoScrollRef.current.raf = requestAnimationFrame(runAutoScroll);
    } else if (velocity === 0) {
      stopAutoScroll();
    }
  };

  useEffect(() => stopAutoScroll, []);

  const resetDrag = (): void => {
    stopAutoScroll();
    setDragIndex(null);
    setDragOverIndex(null);
  };

  const handleDrop = (target: number): void => {
    if (dragIndex !== null && dragIndex !== target) {
      vm.reorderStep(dragIndex, target);
    }
    resetDrag();
  };

  const handleBlockClick = (index: number): void => {
    // Suppress single-select collapse when a range drag just ended.
    if (rangeHasDraggedRef.current) { rangeHasDraggedRef.current = false; return; }
    vm.selectStep(index);
    setSelectedIndices(new Set([index]));
  };

  const handleRangeStart = (index: number): void => {
    rangeDragStartRef.current = index;
    rangeHasDraggedRef.current = false;
    vm.selectStep(index);
    setSelectedIndices(new Set([index]));
  };

  const handleRailMouseMove = (e: React.MouseEvent): void => {
    if (rangeDragStartRef.current === null) return;
    const el = document.elementFromPoint(e.clientX, e.clientY);
    const blockEl = el?.closest('[data-block-index]');
    if (!blockEl) return;
    const idx = Number(blockEl.getAttribute('data-block-index'));
    if (!Number.isFinite(idx)) return;
    const start = rangeDragStartRef.current;
    const lo = Math.min(start, idx), hi = Math.max(start, idx);
    rangeHasDraggedRef.current = lo !== hi;
    setSelectedIndices(new Set(Array.from({ length: hi - lo + 1 }, (_, i) => lo + i)));
  };

  const handleRailMouseUp = (): void => { rangeDragStartRef.current = null; };

  const handleRailKeyDown = (e: React.KeyboardEvent): void => {
    if (e.key === 'Delete' && selectedIndices.size > 0) {
      vm.removeSteps([...selectedIndices]);
      setSelectedIndices(new Set());
    } else if (e.key === 'a' && (e.ctrlKey || e.metaKey)) {
      e.preventDefault();
      if (vm.draft) setSelectedIndices(new Set(vm.draft.steps.map((_, i) => i)));
    }
  };

  const confirmDelete = async (): Promise<void> => {
    if (!vm.draft) return;
    const ok = await NativeDialogs.confirm({
      title: t('settings.macros.confirmDelete.title'),
      message: t('settings.macros.confirmDelete.message', { name: vm.draft.name }),
      okLabel: t('settings.macros.confirmDelete.ok'),
      cancelLabel: t('settings.macros.confirmDelete.cancel'),
    });
    if (ok) vm.deleteActive();
  };

  return (
    <Box sx={styles.page}>
      <Box sx={styles.pageHeader}>
        <Typography sx={styles.pageEyebrow}>{t('macroPage.eyebrow')}</Typography>
        <Typography component="h1" sx={styles.pageTitle}>{t('settings.macros.title')}</Typography>
        <Typography sx={styles.pageHint}>{t('settings.macros.hint')}</Typography>
      </Box>

      {vm.error && <Typography sx={styles.errorText}>{t('settings.macros.error', { message: vm.error })}</Typography>}

      <Box sx={styles.root}>
        <Box sx={styles.panel}>
          <Box sx={styles.panelHeader}>
            <Typography sx={styles.panelTitle}>{t('settings.macros.list.title')}</Typography>
            <Box sx={{ display: 'flex', gap: 0.75 }}>
              <IconTextButton styles={styles} onClick={vm.importMacros} disabled={vm.busy} label={t('settings.macros.actions.import')}>
                <FileUploadRoundedIcon fontSize="small" />
              </IconTextButton>
              <IconTextButton styles={styles} onClick={() => vm.createMacro(t('settings.macros.defaults.name'))} disabled={vm.busy} label={t('settings.macros.actions.new')}>
                <AddRoundedIcon fontSize="small" />
              </IconTextButton>
            </Box>
          </Box>
          <Box sx={styles.list}>
            {vm.macros.length === 0 && <Typography sx={styles.empty}>{t('settings.macros.list.empty')}</Typography>}
            {vm.macros.map((macro) => (
              <Box
                component="button"
                type="button"
                key={macro.id}
                sx={[styles.macroItem, macro.id === vm.activeId ? styles.macroItemActive : false] as SxProps<Theme>}
                onClick={() => vm.selectMacro(macro.id)}
              >
                <Typography sx={styles.macroName}>{macro.name}</Typography>
                <Typography sx={styles.macroMeta}>
                  {macro.triggerAccelerator || t('settings.macros.trigger.none')} · {t('settings.macros.list.steps', { count: macro.stepCount })}
                </Typography>
                <MacroRegStatus macro={macro} styles={styles} />
              </Box>
            ))}
          </Box>
        </Box>

        <Box sx={[styles.panel, styles.editor] as SxProps<Theme>}>
          <Box sx={styles.panelHeader}>
            <Typography sx={styles.panelTitle}>{vm.draft?.name ?? t('settings.macros.editor.emptyTitle')}</Typography>
            <Box sx={{ display: 'flex', gap: 0.75 }}>
              <IconTextButton styles={styles} onClick={vm.saveDraft} disabled={!vm.draft || !vm.dirty || vm.busy} label={t('settings.macros.actions.save')} primary>
                <SaveRoundedIcon fontSize="small" />
              </IconTextButton>
              <IconTextButton styles={styles} onClick={vm.exportActive} disabled={!vm.draft || vm.busy} label={t('settings.macros.actions.export')}>
                <FileDownloadRoundedIcon fontSize="small" />
              </IconTextButton>
              {running ? (
                <IconTextButton styles={styles} onClick={vm.stopRunning} disabled={vm.busy} label={t('settings.macros.actions.stop')} danger>
                  <StopRoundedIcon fontSize="small" />
                </IconTextButton>
              ) : (
                <IconTextButton styles={styles} onClick={vm.runActive} disabled={!vm.draft || vm.busy} label={t('settings.macros.actions.run')} primary>
                  <PlayArrowRoundedIcon fontSize="small" />
                </IconTextButton>
              )}
            </Box>
          </Box>

          <Box sx={styles.toolbar}>
            <AddStepButton kind="keyTap" vm={vm} styles={styles} />
            <AddStepButton kind="keyDown" vm={vm} styles={styles} />
            <AddStepButton kind="keyUp" vm={vm} styles={styles} />
            <AddStepButton kind="delay" vm={vm} styles={styles} />
            <AddStepButton kind="chordTap" vm={vm} styles={styles} />
            <AddStepButton kind="mouseTap" vm={vm} styles={styles} />
            <AddStepButton kind="mouseDown" vm={vm} styles={styles} />
            <AddStepButton kind="mouseUp" vm={vm} styles={styles} />
            <AddStepButton kind="mouseScroll" vm={vm} styles={styles} />
            <AddStepButton kind="mouseMove" vm={vm} styles={styles} />
            <AddStepButton kind="text" vm={vm} styles={styles} />
            {vm.recordState === 'recording' ? (
              <IconTextButton styles={styles} onClick={vm.stopStepRecording} disabled={!vm.draft || vm.busy} label={t('settings.macros.record.stop', { count: vm.recordStepCount })} danger>
                <FiberManualRecordRoundedIcon fontSize="small" />
              </IconTextButton>
            ) : (
              <IconTextButton styles={styles} onClick={vm.startStepRecording} disabled={!vm.draft || vm.busy} label={t('settings.macros.record.start')}>
                <FiberManualRecordRoundedIcon fontSize="small" />
              </IconTextButton>
            )}
            <Box sx={styles.captureDelaysToggle}>
              <Switch
                size="small"
                checked={vm.captureDelays}
                onChange={(e: ChangeEvent<HTMLInputElement>) => vm.setCaptureDelays(e.target.checked)}
                disabled={vm.recordState === 'recording'}
              />
              <Typography sx={styles.captureDelaysLabel}>{t('settings.macros.record.captureDelays')}</Typography>
            </Box>
            <Box sx={styles.captureDelaysToggle}>
              <Switch
                size="small"
                checked={vm.captureMoves}
                onChange={(e: ChangeEvent<HTMLInputElement>) => vm.setCaptureMoves(e.target.checked)}
                disabled={vm.recordState === 'recording'}
              />
              <Typography sx={styles.captureDelaysLabel}>{t('settings.macros.record.captureMoves')}</Typography>
            </Box>
          </Box>

          <Box
            sx={{ ...styles.blockRail, outline: 'none' }}
            ref={railRef}
            tabIndex={0}
            onKeyDown={handleRailKeyDown}
            onMouseMove={handleRailMouseMove}
            onMouseUp={handleRailMouseUp}
            onDragOver={dragIndex !== null ? (e: React.DragEvent) => { e.preventDefault(); updateAutoScroll(e.clientY); } : undefined}
          >
            {!vm.draft || vm.draft.steps.length === 0 ? (
              <Typography sx={styles.empty}>{t('settings.macros.editor.emptyBlocks')}</Typography>
            ) : (
              vm.draft.steps.map((step, index) => (
                <MacroBlock
                  key={`${index}-${step.kind}-${step.keyLabel}`}
                  step={step}
                  index={index}
                  vm={vm}
                  styles={styles}
                  dragging={dragIndex === index}
                  dragOver={dragOverIndex === index && dragIndex !== null && dragIndex !== index}
                  isMultiSelected={selectedIndices.has(index)}
                  onBlockClick={handleBlockClick}
                  onRangeStart={handleRangeStart}
                  onDragStart={() => setDragIndex(index)}
                  onDragEnter={() => setDragOverIndex(index)}
                  onDrop={() => handleDrop(index)}
                  onDragEnd={resetDrag}
                />
              ))
            )}
          </Box>

          <Typography sx={styles.status}>
            {vm.dirty ? t('settings.macros.status.dirty') : t('settings.macros.status.saved')} · {t(`settings.macros.runStates.${vm.runningState}`)}
          </Typography>
        </Box>

        <Box sx={styles.panel}>
          <Box sx={styles.panelHeader}>
            <Typography sx={styles.panelTitle}>{t('settings.macros.properties.title')}</Typography>
          </Box>
          {vm.draft ? (
            <Box sx={styles.sideForm}>
              <LabeledInput label={t('settings.macros.properties.name')} value={vm.draft.name} onChange={(value) => vm.updateDraft({ name: value })} styles={styles} />
              <LabeledInput label={t('settings.macros.properties.description')} value={vm.draft.description} onChange={(value) => vm.updateDraft({ description: value })} styles={styles} multiline />

              <Box sx={styles.field}>
                <Typography sx={styles.label}>{t('settings.macros.properties.trigger')}</Typography>
                <Box component="button" type="button" sx={[styles.button, vm.recordingTrigger ? styles.primaryButton : false] as SxProps<Theme>} onClick={vm.recordingTrigger ? vm.cancelTriggerRecording : vm.startTriggerRecording}>
                  <KeyboardRoundedIcon fontSize="small" />
                  {vm.recordingTrigger ? t('settings.macros.trigger.recording') : vm.draft.triggerAccelerator || t('settings.macros.trigger.none')}
                </Box>
                {vm.recordingHint && <Typography sx={styles.hintText}>{t(`settings.macros.trigger.hints.${vm.recordingHint}`)}</Typography>}
                <Typography sx={styles.hintText}>{t('settings.macros.properties.triggerHint')}</Typography>
              </Box>

              <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
                <Typography sx={styles.label}>{t('settings.macros.properties.enabled')}</Typography>
                <Switch checked={vm.draft.enabled} onChange={(e) => vm.setEnabled(e.target.checked)} />
              </Box>
              {vm.draft.triggerAccelerator && !vm.draft.enabled && (
                <Typography sx={styles.hintText}>{t('settings.macros.regStatus.disabledWithTrigger')}</Typography>
              )}

              <Box sx={styles.field}>
                <Typography sx={styles.label}>{t('settings.macros.properties.interrupt')}</Typography>
                <Box component="select" sx={styles.input} value={vm.draft.interruptPolicy} onChange={(e) => vm.updateDraft({ interruptPolicy: e.target.value })}>
                  <option value="ignore">{t('settings.macros.interrupt.ignore')}</option>
                  <option value="stop-current-and-run">{t('settings.macros.interrupt.restart')}</option>
                </Box>
              </Box>

              <Box sx={styles.field}>
                <Typography sx={styles.label}>{t('settings.macros.properties.repeatMode')}</Typography>
                <Box component="select" sx={styles.input} value={vm.draft.repeatMode} onChange={(e) => vm.updateDraft({ repeatMode: e.target.value as MacroRepeatMode })}>
                  <option value="once">{t('settings.macros.repeatModes.once')}</option>
                  <option value="count">{t('settings.macros.repeatModes.count')}</option>
                  <option value="hold">{t('settings.macros.repeatModes.hold')}</option>
                  <option value="toggle">{t('settings.macros.repeatModes.toggle')}</option>
                </Box>
              </Box>

              {vm.draft.repeatMode === 'count' && (
                <LabeledInput label={t('settings.macros.properties.repeatCount')} value={String(vm.draft.repeatCount)} onChange={(value) => vm.updateDraft({ repeatCount: Math.max(1, Number(value || 1)) })} styles={styles} type="number" />
              )}

              {vm.draft.repeatMode !== 'once' && (
                <LabeledInput label={t('settings.macros.properties.repeatInterval')} value={String(vm.draft.repeatIntervalMs)} onChange={(value) => vm.updateDraft({ repeatIntervalMs: Math.max(0, Number(value || 0)) })} styles={styles} type="number" />
              )}

              <IconTextButton styles={styles} onClick={confirmDelete} disabled={!vm.activeId || vm.busy} label={t('settings.macros.actions.delete')} danger>
                <DeleteOutlineRoundedIcon fontSize="small" />
              </IconTextButton>

              <StepProperties vm={vm} step={selectedStep} styles={styles} />
            </Box>
          ) : (
            <Typography sx={styles.empty}>{t('settings.macros.properties.empty')}</Typography>
          )}
        </Box>
      </Box>
    </Box>
  );
};

type Styles = ReturnType<typeof macroPageStyles>;

const isMouseKind = (kind: MacroStepKind): boolean =>
  kind === 'mouseTap' || kind === 'mouseDown' || kind === 'mouseUp' || kind === 'mouseScroll';

// Wheel direction ids mirror the keysim MouseWheel* catalogue (6..9).
const isWheelButton = (virtualKey: number): boolean => virtualKey >= 6 && virtualKey <= 9;

const AddStepButton = ({ kind, vm, styles }: { kind: MacroStepKind; vm: UseMacroPageResult; styles: Styles }) => {
  const t = useT();
  const icon = kind === 'delay' ? <TimerRoundedIcon fontSize="small" /> : kind === 'text' ? <TextFieldsRoundedIcon fontSize="small" /> : kind === 'mouseScroll' ? <SwapVertRoundedIcon fontSize="small" /> : kind === 'mouseMove' ? <OpenWithRoundedIcon fontSize="small" /> : isMouseKind(kind) ? <MouseRoundedIcon fontSize="small" /> : <KeyboardRoundedIcon fontSize="small" />;
  return (
    <IconTextButton styles={styles} onClick={() => vm.addStep(kind)} disabled={!vm.draft || vm.busy || vm.recordState === 'recording'} label={t(`settings.macros.stepKinds.${kind}.add`)}>
      {icon}
    </IconTextButton>
  );
};

const MacroBlock = ({ step, index, vm, styles, dragging, dragOver, isMultiSelected, onBlockClick, onRangeStart, onDragStart, onDragEnter, onDrop, onDragEnd }: {
  step: MacroStep;
  index: number;
  vm: UseMacroPageResult;
  styles: Styles;
  dragging: boolean;
  dragOver: boolean;
  isMultiSelected: boolean;
  onBlockClick: (index: number) => void;
  onRangeStart: (index: number) => void;
  onDragStart: () => void;
  onDragEnter: () => void;
  onDrop: () => void;
  onDragEnd: () => void;
}) => {
  const t = useT();
  const isRunning = vm.runningMacroId === vm.activeId && vm.runningStepIndex === index && vm.runningState === 'running';
  return (
    <Box
      component="button"
      type="button"
      data-block-index={index}
      sx={[styles.block, isMultiSelected ? styles.blockSelected : false, isRunning ? styles.blockRunning : false, dragging ? styles.blockDragging : false, dragOver ? styles.blockDragOver : false] as SxProps<Theme>}
      onClick={() => onBlockClick(index)}
      onMouseDown={(e: React.MouseEvent) => {
        const rect = e.currentTarget.getBoundingClientRect();
        if (e.button === 0 && e.clientX > rect.left + rect.width / 2) {
          e.preventDefault();
          onRangeStart(index);
        }
      }}
      onDragOver={(e: React.DragEvent) => { e.preventDefault(); }}
      onDragEnter={(e: React.DragEvent) => { e.preventDefault(); onDragEnter(); }}
      onDrop={(e: React.DragEvent) => { e.preventDefault(); onDrop(); }}
    >
      <Typography sx={styles.blockIndex}>{index + 1}</Typography>
      <Box
        sx={styles.blockIcon}
        draggable
        onClick={(e: React.MouseEvent) => e.stopPropagation()}
        onDragStart={(e: React.DragEvent) => { e.dataTransfer.effectAllowed = 'move'; e.dataTransfer.setData('text/plain', String(index)); onDragStart(); }}
        onDragEnd={onDragEnd}
        aria-label={t('settings.macros.block.drag')}
        title={t('settings.macros.block.drag')}
      >
        {step.kind === 'delay' ? <TimerRoundedIcon fontSize="small" /> : step.kind === 'text' ? <TextFieldsRoundedIcon fontSize="small" /> : step.kind === 'mouseMove' ? <OpenWithRoundedIcon fontSize="small" /> : isMouseKind(step.kind) ? <MouseRoundedIcon fontSize="small" /> : <AccountTreeRoundedIcon fontSize="small" />}
      </Box>
      <Box sx={styles.blockBadges}>
        {step.kind === 'delay' ? (
          <>
            <EditableDelayBadge
              value={step.waitMs}
              onCommit={(next) => vm.updateStep(index, { waitMs: next })}
              styles={styles}
              title={t('settings.macros.block.editDelay')}
            />
            <Box sx={styles.blockBadge}>{t('settings.macros.units.ms')}</Box>
          </>
        ) : (
          stepBadges(step, t).map((badge, i) => (
            <Box key={i} sx={styles.blockBadge}>{badge}</Box>
          ))
        )}
      </Box>
      <Typography sx={styles.blockTitle}>{t(`settings.macros.stepKinds.${step.kind}.label`)}</Typography>
      <Box sx={styles.blockActions} onClick={(e) => e.stopPropagation()}>
        <SmallIconButton styles={styles} onClick={() => vm.moveStep(index, -1)} disabled={index === 0} label={t('settings.macros.block.moveUp')}>
          <ArrowUpwardRoundedIcon fontSize="small" />
        </SmallIconButton>
        <SmallIconButton styles={styles} onClick={() => vm.moveStep(index, 1)} disabled={!vm.draft || index >= vm.draft.steps.length - 1} label={t('settings.macros.block.moveDown')}>
          <ArrowDownwardRoundedIcon fontSize="small" />
        </SmallIconButton>
        <SmallIconButton styles={styles} onClick={() => vm.duplicateStep(index)} label={t('settings.macros.block.duplicate')}>
          <ContentCopyRoundedIcon fontSize="small" />
        </SmallIconButton>
        <SmallIconButton styles={styles} onClick={() => vm.removeStep(index)} label={t('settings.macros.block.delete')}>
          <DeleteOutlineRoundedIcon fontSize="small" />
        </SmallIconButton>
      </Box>
    </Box>
  );
};

// EditableDelayBadge renders the delay value badge. A double-click swaps the
// static chip for an inline number input so users can retype the millisecond
// value directly on the timeline. Editing is committed on Enter / blur and
// abandoned on Escape; clicks are stopped from bubbling to the block's select
// / drag handlers.
const EditableDelayBadge = ({ value, onCommit, styles, title }: { value: number; onCommit: (next: number) => void; styles: Styles; title: string }) => {
  const [editing, setEditing] = useState(false);
  const [text, setText] = useState(String(value));
  const inputRef = useRef<HTMLInputElement | null>(null);

  useEffect(() => {
    if (editing) {
      inputRef.current?.focus();
      inputRef.current?.select();
    }
  }, [editing]);

  const beginEdit = (): void => {
    setText(String(value));
    setEditing(true);
  };

  const commit = (): void => {
    const next = Math.max(0, Math.trunc(Number(text)));
    if (Number.isFinite(next) && next !== value) onCommit(next);
    setEditing(false);
  };

  if (editing) {
    return (
      <Box
        component="input"
        type="number"
        ref={inputRef}
        sx={[styles.blockBadge, styles.blockBadgeInput] as SxProps<Theme>}
        value={text}
        onClick={(e: React.MouseEvent) => e.stopPropagation()}
        onChange={(e: ChangeEvent<HTMLInputElement>) => setText(e.target.value)}
        onBlur={commit}
        onKeyDown={(e: React.KeyboardEvent) => {
          e.stopPropagation();
          if (e.key === 'Enter') commit();
          else if (e.key === 'Escape') setEditing(false);
        }}
      />
    );
  }

  return (
    <Box
      sx={[styles.blockBadge, styles.blockBadgeEditable] as SxProps<Theme>}
      title={title}
      onClick={(e: React.MouseEvent) => e.stopPropagation()}
      onDoubleClick={(e: React.MouseEvent) => { e.stopPropagation(); beginEdit(); }}
    >
      {value}
    </Box>
  );
};

const StepProperties = ({ vm, step, styles }: { vm: UseMacroPageResult; step: MacroStep | null; styles: Styles }) => {
  const t = useT();
  if (!step || vm.selectedStepIndex < 0) {
    return <Typography sx={styles.empty}>{t('settings.macros.properties.noStep')}</Typography>;
  }
  const mouse = isMouseKind(step.kind);
  const isScroll = step.kind === 'mouseScroll';
  // Scroll steps choose among wheel directions (ids 6..9); click steps choose
  // among the five physical buttons (ids 1..5).
  const mouseKeys = vm.keys.filter((k) => k.deviceKind === 'mouse' && (isScroll ? isWheelButton(k.virtualKey) : !isWheelButton(k.virtualKey)));
  const nonModifierKeys = vm.keys.filter((k) => !k.modifier && k.deviceKind !== 'mouse');
  const modifierKeys = vm.keys.filter((k) => k.modifier);
  const keyOptions = mouse ? mouseKeys : nonModifierKeys;
  const updateKey = (label: string): void => {
    const key = keyOptions.find((k) => k.label === label);
    if (!key) return;
    vm.updateStep(vm.selectedStepIndex, { keyLabel: key.label, virtualKey: key.virtualKey, scanCode: key.scanCode, deviceKind: key.deviceKind });
  };
  // Chord modifiers are a set: toggling a modifier adds/removes it from the
  // persisted array while preserving the other selected modifiers. The backend
  // requires at least one modifier, so the last one cannot be removed.
  const selectedModifiers = parseModifierLabels(step.modifierKeysJson);
  const toggleModifier = (label: string): void => {
    const key = modifierKeys.find((k) => k.label === label);
    if (!key) return;
    const isSelected = selectedModifiers.includes(label);
    if (isSelected && selectedModifiers.length <= 1) return; // keep at least one
    const next = modifierKeys
      .filter((k) => (k.label === label ? !isSelected : selectedModifiers.includes(k.label)))
      .map((k) => ({ label: k.label, virtualKey: k.virtualKey, scanCode: k.scanCode }));
    vm.updateStep(vm.selectedStepIndex, { modifierKeysJson: JSON.stringify(next) });
  };
  return (
    <Box sx={styles.field}>
      <Typography sx={styles.panelTitle}>{t('settings.macros.properties.stepTitle')}</Typography>
      {step.kind === 'text' && (
        <>
          <LabeledInput
            label={t('settings.macros.properties.textLabel')}
            value={textPayloadValue(step.payloadJson)}
            onChange={(value) => vm.updateStep(vm.selectedStepIndex, { payloadJson: withTextPayload(step.payloadJson, value) })}
            styles={styles}
            multiline
            placeholder={t('settings.macros.properties.textPlaceholder')}
          />
          <LabeledInput
            label={t('settings.macros.properties.textDelay')}
            value={String(textDelayValue(step.payloadJson))}
            onChange={(value) => vm.updateStep(vm.selectedStepIndex, { payloadJson: withTextDelay(step.payloadJson, Math.max(0, Number(value || 0))) })}
            styles={styles}
            type="number"
          />
        </>
      )}
      {step.kind === 'mouseMove' && (
        <>
          <LabeledInput
            label={t('settings.macros.properties.moveDx')}
            value={String(movePayloadValue(step.payloadJson).dx)}
            onChange={(value) => vm.updateStep(vm.selectedStepIndex, { payloadJson: withMoveOffset(step.payloadJson, 'dx', Number(value || 0)) })}
            styles={styles}
            type="number"
          />
          <LabeledInput
            label={t('settings.macros.properties.moveDy')}
            value={String(movePayloadValue(step.payloadJson).dy)}
            onChange={(value) => vm.updateStep(vm.selectedStepIndex, { payloadJson: withMoveOffset(step.payloadJson, 'dy', Number(value || 0)) })}
            styles={styles}
            type="number"
          />
          <LabeledInput
            label={t('settings.macros.properties.moveDuration')}
            value={String(step.durationMs)}
            onChange={(value) => vm.updateStep(vm.selectedStepIndex, { durationMs: Math.max(0, Number(value || 0)) })}
            styles={styles}
            type="number"
          />
          <LabeledInput
            label={t('settings.macros.properties.moveJitter')}
            value={String(movePayloadValue(step.payloadJson).jitter)}
            onChange={(value) => vm.updateStep(vm.selectedStepIndex, { payloadJson: withMoveJitter(step.payloadJson, Number(value || 0)) })}
            styles={styles}
            type="number"
          />
          <Typography sx={styles.hintText}>{t('settings.macros.properties.moveHint')}</Typography>
        </>
      )}
      {step.kind !== 'delay' && step.kind !== 'text' && step.kind !== 'mouseMove' && (
        <Box sx={styles.field}>
          <Typography sx={styles.label}>{isScroll ? t('settings.macros.properties.direction') : mouse ? t('settings.macros.properties.button') : t('settings.macros.properties.key')}</Typography>
          <Box component="select" sx={styles.input} value={step.keyLabel} onChange={(e) => updateKey(e.target.value)}>
            {keyOptions.map((key) => <option key={key.label} value={key.label}>{key.label}</option>)}
          </Box>
        </Box>
      )}
      {step.kind === 'chordTap' && (
        <Box sx={styles.field}>
          <Typography sx={styles.label}>{t('settings.macros.properties.modifier')}</Typography>
          <Box sx={{ display: 'flex', flexWrap: 'wrap', gap: 0.5 }}>
            {modifierKeys.map((key) => {
              const active = selectedModifiers.includes(key.label);
              return (
                <Box
                  component="button"
                  type="button"
                  key={key.label}
                  sx={[styles.button, active ? styles.primaryButton : false] as SxProps<Theme>}
                  onClick={() => toggleModifier(key.label)}
                >
                  {key.label}
                </Box>
              );
            })}
          </Box>
          <Typography sx={styles.hintText}>{t('settings.macros.properties.modifierHint')}</Typography>
        </Box>
      )}
      {(step.kind === 'keyTap' || step.kind === 'chordTap' || step.kind === 'mouseTap') && (
        <LabeledInput label={t('settings.macros.properties.duration')} value={String(step.durationMs)} onChange={(value) => vm.updateStep(vm.selectedStepIndex, { durationMs: Number(value || 0) })} styles={styles} type="number" />
      )}
      {step.kind === 'delay' && (
        <LabeledInput label={t('settings.macros.properties.wait')} value={String(step.waitMs)} onChange={(value) => vm.updateStep(vm.selectedStepIndex, { waitMs: Number(value || 0) })} styles={styles} type="number" />
      )}
    </Box>
  );
};

const MacroRegStatus = ({ macro, styles }: { macro: MacroSummary; styles: Styles }) => {
  const t = useT();
  // Surface why a macro will / will not fire globally, for every state — not
  // just failures. A trigger set on a disabled macro never reaches RegisterHotKey,
  // which is the most common "my hotkey does nothing" cause.
  if (macro.enabled && macro.errorCode) {
    return <Typography sx={styles.errorText}>{t(`settings.macros.hotkeyErrors.${macro.errorCode}`)}</Typography>;
  }
  if (!macro.triggerAccelerator) {
    return <Typography sx={styles.statusMuted}>{t('settings.macros.regStatus.noTrigger')}</Typography>;
  }
  if (!macro.enabled) {
    return <Typography sx={styles.statusMuted}>{t('settings.macros.regStatus.disabledWithTrigger')}</Typography>;
  }
  if (macro.registered) {
    return <Typography sx={styles.statusOk}>{t('settings.macros.regStatus.active')}</Typography>;
  }
  return <Typography sx={styles.statusMuted}>{t('settings.macros.regStatus.registering')}</Typography>;
};

const LabeledInput = ({ label, value, onChange, styles, multiline = false, type = 'text', placeholder }: { label: string; value: string; onChange: (value: string) => void; styles: Styles; multiline?: boolean; type?: string; placeholder?: string }) => {
  const handleChange = (event: ChangeEvent<HTMLInputElement | HTMLTextAreaElement>): void => {
    onChange(event.target.value);
  };
  return (
    <Box sx={styles.field}>
      <Typography sx={styles.label}>{label}</Typography>
      <Box component={multiline ? 'textarea' : 'input'} type={type} placeholder={placeholder} sx={[styles.input, multiline ? styles.textarea : false] as SxProps<Theme>} value={value} onChange={handleChange} />
    </Box>
  );
};

const IconTextButton = ({ children, label, onClick, disabled, styles, primary, danger }: { children: ReactNode; label: string; onClick: () => void; disabled?: boolean; styles: Styles; primary?: boolean; danger?: boolean }) => (
  <Box component="button" type="button" sx={[styles.button, primary ? styles.primaryButton : false, danger ? styles.dangerButton : false] as SxProps<Theme>} onClick={onClick} disabled={disabled}>
    {children}
    {label}
  </Box>
);

const SmallIconButton = ({ children, label, onClick, disabled, styles }: { children: React.ReactNode; label: string; onClick: () => void; disabled?: boolean; styles: Styles }) => (
  <Box component="button" type="button" sx={{ ...styles.button, minWidth: 30, width: 30, height: 30, p: 0 }} aria-label={label} title={label} onClick={onClick} disabled={disabled}>
    {children}
  </Box>
);

// stepBadges renders the most salient parameters of a step as small chips shown
// to the left of the centered title (e.g. the key label, or a delay's value and
// unit), replacing the old grey subtitle line.
const stepBadges = (step: MacroStep, t: ReturnType<typeof useT>): string[] => {
  switch (step.kind) {
    case 'delay':
      return [String(step.waitMs), t('settings.macros.units.ms')];
    case 'keyTap':
    case 'keyDown':
    case 'keyUp':
    case 'mouseTap':
    case 'mouseDown':
    case 'mouseUp':
    case 'mouseScroll':
      return [step.keyLabel];
    case 'chordTap':
      return [...joinModifierLabels(step.modifierKeysJson).split(' + '), step.keyLabel];
    case 'mouseMove': {
      const { dx, dy } = movePayloadValue(step.payloadJson);
      return [`${dx}`, `${dy}`];
    }
    case 'text': {
      const text = textPayloadValue(step.payloadJson);
      return [text.length > 10 ? `${text.slice(0, 10)}…` : text || '—'];
    }
    default:
      return [];
  }
};

// Text payload helpers: read/write the { text, perCharDelayMs } JSON stored in
// the reused payloadJson column without clobbering the other field.
const parseTextPayload = (raw: string): { text: string; perCharDelayMs: number } => {
  try {
    const obj = JSON.parse(raw || '{}') as { text?: unknown; perCharDelayMs?: unknown };
    return { text: typeof obj.text === 'string' ? obj.text : '', perCharDelayMs: Number(obj.perCharDelayMs ?? 0) || 0 };
  } catch {
    return { text: '', perCharDelayMs: 0 };
  }
};

const textPayloadValue = (raw: string): string => parseTextPayload(raw).text;

const textDelayValue = (raw: string): number => parseTextPayload(raw).perCharDelayMs;

const withTextPayload = (raw: string, text: string): string => {
  const current = parseTextPayload(raw);
  return JSON.stringify({ text, perCharDelayMs: current.perCharDelayMs });
};

const withTextDelay = (raw: string, perCharDelayMs: number): string => {
  const current = parseTextPayload(raw);
  return JSON.stringify({ text: current.text, perCharDelayMs });
};

// Move payload helpers: read/write the { dx, dy, jitter } JSON stored in the
// reused payloadJson column. Offsets and jitter are clamped to the backend
// bounds so the editor cannot persist a value the planner would reject.
const maxMoveDelta = 10_000;
const maxMoveJitter = 500;

const movePayloadValue = (raw: string): { dx: number; dy: number; jitter: number } => {
  try {
    const obj = JSON.parse(raw || '{}') as { dx?: unknown; dy?: unknown; jitter?: unknown };
    return { dx: Number(obj.dx ?? 0) || 0, dy: Number(obj.dy ?? 0) || 0, jitter: Number(obj.jitter ?? 0) || 0 };
  } catch {
    return { dx: 0, dy: 0, jitter: 0 };
  }
};

const clampMove = (value: number): number => Math.max(-maxMoveDelta, Math.min(maxMoveDelta, Math.trunc(value)));

const clampJitter = (value: number): number => Math.max(0, Math.min(maxMoveJitter, Math.trunc(value)));

const withMoveOffset = (raw: string, axis: 'dx' | 'dy', value: number): string => {
  const current = movePayloadValue(raw);
  return JSON.stringify({ ...current, [axis]: clampMove(value) });
};

const withMoveJitter = (raw: string, value: number): string => {
  const current = movePayloadValue(raw);
  return JSON.stringify({ ...current, jitter: clampJitter(value) });
};

// parseModifierLabels reads the persisted modifier array into a label list.
// Chord steps store an array of {label, virtualKey, scanCode}; the editor and
// description join them so multi-modifier combos (Ctrl+Shift+S) render fully.
const parseModifierLabels = (raw: string): string[] => {
  try {
    const rows = JSON.parse(raw) as Array<{ label?: string }>;
    return rows.map((r) => r.label ?? '').filter(Boolean);
  } catch {
    return [];
  }
};

const joinModifierLabels = (raw: string): string => {
  const labels = parseModifierLabels(raw);
  return labels.length > 0 ? labels.join(' + ') : 'Ctrl';
};
