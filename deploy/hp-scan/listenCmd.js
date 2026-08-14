/*
 * Derivative of node-hp-scan-to 1.8.0 (MIT License).
 *
 * Upstream: https://github.com/manuc66/node-hp-scan-to
 * Author: Emmanuel Counasse (https://github.com/manuc66)
 * Tag / image: v1.8.0 (compiled command module bind-mounted over
 *   /app/commands/listenCmd.js in the digest-pinned Docker image).
 *
 * Copyright (c) 2022 Emmanuel Counasse
 *
 * Sonix modifications (same MIT terms):
 *   1. Emulated-duplex back pass inherits scanToPdf from the front pass.
 *   2. Flush pending odds when switching to the simplex "Sonix" target.
 * Drop this overlay when the pinned image includes those fixes.
 * Attribution: THIRD-PARTY-NOTICES.md
 *
 * MIT License
 *
 * Permission is hereby granted, free of charge, to any person obtaining a copy
 * of this software and associated documentation files (the "Software"), to deal
 * in the Software without restriction, including without limitation the rights
 * to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
 * copies of the Software, and to permit persons to whom the Software is
 * furnished to do so, subject to the following conditions:
 *
 * The above copyright notice and this permission notice shall be included in all
 * copies or substantial portions of the Software.
 *
 * THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
 * IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
 * FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
 * AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
 * LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
 * OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
 * SOFTWARE.
 */
"use strict";
var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
Object.defineProperty(exports, "__esModule", { value: true });
exports.listenCmd = listenCmd;
const HPApi_1 = __importDefault(require("../HPApi"));
const readDeviceCapabilities_1 = require("../readDeviceCapabilities");
const listening_1 = require("../listening");
const scanProcessing_1 = require("../scanProcessing");
const postProcessing_1 = require("../postProcessing");
const PathHelper_1 = __importDefault(require("../PathHelper"));
const delay_1 = require("../delay");
const duplexMode_1 = require("../type/duplexMode");
const targetDuplexMode_1 = require("../type/targetDuplexMode");
const pageCountingStrategy_1 = require("../type/pageCountingStrategy");
let iteration = 0;
async function listenCmd(registrationConfigs, scanConfig, deviceUpPollingInterval) {
    // first make sure the device is reachable
    await HPApi_1.default.waitDeviceUp(deviceUpPollingInterval);
    let deviceUp = true;
    const folder = await PathHelper_1.default.getTargetFolder(scanConfig.directoryConfig.directory);
    const tempFolder = await PathHelper_1.default.getTempFolder(scanConfig.directoryConfig.tempDirectory);
    const deviceCapabilities = await (0, readDeviceCapabilities_1.readDeviceCapabilities)();
    let scanCount = 0;
    let keepActive = true;
    let errorCount = 0;
    let lastScanTarget = undefined;
    let lastDuplexMode = duplexMode_1.DuplexMode.Simplex;
    let frontOfDoubleSidedScanContext = null;
    while (keepActive) {
        iteration++;
        console.log(`Running iteration: ${iteration} - errorCount: ${errorCount}`);
        try {
            const selectedScanTarget = await (0, listening_1.waitScanEvent)(deviceCapabilities, registrationConfigs);
            let proceedToScan = true;
            if (selectedScanTarget.event.compEventURI) {
                proceedToScan = await (0, listening_1.waitScanRequest)(selectedScanTarget.event.compEventURI);
            }
            let destination = null;
            if (!proceedToScan) {
                console.log("Device state doesn't match expectations - Unable to proceed with scan, skipping.");
            }
            else {
                destination = await (0, scanProcessing_1.tryGetDestination)(selectedScanTarget.event);
                if (!destination) {
                    console.log("No shortcut selected - Impossible to proceed with scan, skipping.");
                }
            }
            if (destination) {
                console.log("Selected shortcut: " + destination.shortcut);
                const { duplexMode, targetDuplexMode } = determineDuplexModes(destination, selectedScanTarget, lastDuplexMode, lastScanTarget);
                // Sonix patch (1.8.0): flush pending odds when leaving duplex for simplex/hardware-duplex.
                // Stock checked selectedScanTarget.isDuplexSingleSide (never true for "Sonix"), so flush never ran.
                // Match upstream master: last target was duplex front, new mode is Simplex or hardware Duplex.
                if (lastScanTarget != null &&
                    frontOfDoubleSidedScanContext != null &&
                    lastScanTarget.isDuplexSingleSide &&
                    lastDuplexMode === duplexMode_1.DuplexMode.FrontOfDoubleSided &&
                    (duplexMode === duplexMode_1.DuplexMode.Simplex || duplexMode === duplexMode_1.DuplexMode.Duplex)) {
                    await processFinishedPartialDuplexScan(lastScanTarget, selectedScanTarget, scanCount, frontOfDoubleSidedScanContext);
                    frontOfDoubleSidedScanContext = null;
                }
                // Sonix patch (1.8.0): pass front context so back sides inherit scanToPdf.
                // Upstream 1.8.0 left scanToPdf=false on backs → pages written to DIR (inbox)
                // instead of TEMP_DIR; Sonix imported each JPEG before merge (ENOENT race).
                // Same fix as upstream master setupScanParameters(..., frontOfDoubleSidedScanContext).
                const { pageCountingStrategy, scanToPdf, scanDate, scanCount: newScanCount, } = await setupScanParameters(duplexMode, targetDuplexMode, destination, scanCount, folder, scanConfig, frontOfDoubleSidedScanContext);
                scanCount = newScanCount;
                const scanJobContent = await (0, scanProcessing_1.saveScanFromEvent)(selectedScanTarget, folder, tempFolder, scanCount, deviceCapabilities, scanConfig, targetDuplexMode == targetDuplexMode_1.TargetDuplexMode.Duplex, scanToPdf, pageCountingStrategy);
                frontOfDoubleSidedScanContext = await handleScanResult(duplexMode, frontOfDoubleSidedScanContext, scanConfig, folder, tempFolder, scanCount, scanJobContent, scanDate, scanToPdf);
                lastScanTarget = selectedScanTarget;
                lastDuplexMode = duplexMode;
            }
        }
        catch (e) {
            if (await HPApi_1.default.isAlive()) {
                console.log(e);
                errorCount++;
            }
            else {
                if (HPApi_1.default.isDebug()) {
                    console.log(e);
                }
                deviceUp = false;
            }
        }
        if (errorCount === 50) {
            keepActive = false;
        }
        if (!deviceUp) {
            await HPApi_1.default.waitDeviceUp(deviceUpPollingInterval);
        }
        else {
            await (0, delay_1.delay)(1000);
        }
    }
}
async function handleScanResult(duplexMode, frontOfDoubleSidedScanContext, scanConfig, folder, tempFolder, scanCount, scanJobContent, scanDate, scanToPdf) {
    if (duplexMode == duplexMode_1.DuplexMode.FrontOfDoubleSided) {
        frontOfDoubleSidedScanContext = {
            scanConfig,
            folder,
            tempFolder,
            scanCount,
            scanJobContent,
            scanDate,
            scanToPdf
        };
    }
    else {
        let finalScanJobContent;
        if (duplexMode == duplexMode_1.DuplexMode.BackOfDoubleSided) {
            finalScanJobContent = assembleDuplexScan(frontOfDoubleSidedScanContext, scanJobContent);
        }
        else {
            finalScanJobContent = scanJobContent;
        }
        await (0, postProcessing_1.postProcessing)(scanConfig, folder, tempFolder, scanCount, finalScanJobContent, scanDate, scanToPdf);
    }
    return frontOfDoubleSidedScanContext;
}
function determineDuplexModes(destination, selectedScanTarget, previousDuplexMode, lastScanTarget) {
    const isDuplex = destination.scanPlexMode != null && destination.scanPlexMode != "Simplex";
    let duplexMode;
    let targetDuplexMode;
    if (isDuplex) {
        targetDuplexMode = targetDuplexMode_1.TargetDuplexMode.Duplex;
        duplexMode = duplexMode_1.DuplexMode.Duplex;
    }
    else if (selectedScanTarget.isDuplexSingleSide) {
        targetDuplexMode = targetDuplexMode_1.TargetDuplexMode.EmulatedDuplex;
        if (lastScanTarget != null &&
            selectedScanTarget.resourceURI === lastScanTarget.resourceURI &&
            previousDuplexMode !== duplexMode_1.DuplexMode.BackOfDoubleSided) {
            duplexMode = duplexMode_1.DuplexMode.BackOfDoubleSided;
        }
        else {
            duplexMode = duplexMode_1.DuplexMode.FrontOfDoubleSided;
        }
    }
    else {
        targetDuplexMode = targetDuplexMode_1.TargetDuplexMode.Simplex;
        duplexMode = duplexMode_1.DuplexMode.Simplex;
    }
    return { duplexMode, targetDuplexMode };
}
function assembleEmulatedDoubleSideScan(previousScanContent, scanJobContent) {
    const frontContent = previousScanContent.elements;
    const backContent = scanJobContent.elements;
    const duplexScanJobContent = { elements: [] };
    for (let i = 0; i < Math.max(frontContent.length, backContent.length); i++) {
        if (i < frontContent.length) {
            duplexScanJobContent.elements.push(frontContent[i]);
        }
        if (i < backContent.length) {
            duplexScanJobContent.elements.push(backContent[i]);
        }
    }
    return duplexScanJobContent;
}
async function setupScanParameters(duplexMode, targetDuplexMode, destination, scanCount, folder, scanConfig, frontOfDoubleSidedScanContext = null) {
    let pageCountingStrategy;
    let scanToPdf = false;
    let scanDate = new Date();
    if (duplexMode == duplexMode_1.DuplexMode.Duplex) {
        console.log(`Destination ScanPlexMode is : ${targetDuplexMode}`);
        pageCountingStrategy = pageCountingStrategy_1.PageCountingStrategy.Normal;
        scanToPdf = (0, scanProcessing_1.isPdf)(destination);
        scanDate = new Date();
        scanCount = await PathHelper_1.default.getNextScanNumber(folder, scanCount, scanConfig.directoryConfig.filePattern);
        console.log(`Scan event captured, saving scan #${scanCount}`);
    }
    else if (targetDuplexMode == targetDuplexMode_1.TargetDuplexMode.EmulatedDuplex) {
        if (duplexMode == duplexMode_1.DuplexMode.FrontOfDoubleSided) {
            console.log(`Destination ScanPlexMode is : ${targetDuplexMode}`);
            pageCountingStrategy = pageCountingStrategy_1.PageCountingStrategy.OddOnly;
            scanToPdf = (0, scanProcessing_1.isPdf)(destination);
            scanDate = new Date();
            scanCount = await PathHelper_1.default.getNextScanNumber(folder, scanCount, scanConfig.directoryConfig.filePattern);
            console.log(`Scan event captured, saving front sides of scan #${scanCount}`);
        }
        else {
            console.log(`Destination ScanPlexMode is : ${targetDuplexMode}`);
            pageCountingStrategy = pageCountingStrategy_1.PageCountingStrategy.EvenOnly;
            // Inherit PDF intent + scan identity from the front pass (upstream master fix).
            scanToPdf = (frontOfDoubleSidedScanContext && frontOfDoubleSidedScanContext.scanToPdf != null)
                ? frontOfDoubleSidedScanContext.scanToPdf
                : (0, scanProcessing_1.isPdf)(destination);
            scanDate = (frontOfDoubleSidedScanContext && frontOfDoubleSidedScanContext.scanDate)
                ? frontOfDoubleSidedScanContext.scanDate
                : new Date();
            scanCount = (frontOfDoubleSidedScanContext && frontOfDoubleSidedScanContext.scanCount != null)
                ? frontOfDoubleSidedScanContext.scanCount
                : scanCount;
            console.log(`Scan event captured, saving back sides of scan #${scanCount}`);
        }
    }
    else {
        console.log(`Destination ScanPlexMode is : ${targetDuplexMode}`);
        pageCountingStrategy = pageCountingStrategy_1.PageCountingStrategy.Normal;
        scanToPdf = (0, scanProcessing_1.isPdf)(destination);
        scanDate = new Date();
        scanCount = await PathHelper_1.default.getNextScanNumber(folder, scanCount, scanConfig.directoryConfig.filePattern);
        console.log(`Scan event captured, saving scan #${scanCount}`);
    }
    return { pageCountingStrategy, scanToPdf, scanDate, scanCount };
}
async function processFinishedPartialDuplexScan(lastScanTarget, selectedScanTarget, scanCount, frontOfDoubleSidedScanContext) {
    console.log(`Scan target changed from ${lastScanTarget.label} to ${selectedScanTarget.label}, saving scan #${scanCount} before processing`);
    await (0, postProcessing_1.postProcessing)(frontOfDoubleSidedScanContext.scanConfig, frontOfDoubleSidedScanContext.folder, frontOfDoubleSidedScanContext.tempFolder, frontOfDoubleSidedScanContext.scanCount, frontOfDoubleSidedScanContext.scanJobContent, frontOfDoubleSidedScanContext.scanDate, frontOfDoubleSidedScanContext.scanToPdf);
}
function assembleDuplexScan(frontOfDoubleSidedScanContext, scanJobContent) {
    console.log("Emulated duplex scan completed; front and back pages are being assembled");
    return assembleEmulatedDoubleSideScan(frontOfDoubleSidedScanContext?.scanJobContent ?? { elements: [] }, scanJobContent);
}
//# sourceMappingURL=listenCmd.js.map