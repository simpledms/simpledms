// Subsets the Material Symbols icon font to the icons SimpleDMS uses. The full font contains
// thousands of icons (about 390 KB), which browsers would otherwise download on the first visit.
//
// Run `npm run subset-icons` after using a new icon. Icon names are collected from string
// literals (Go, JS) and template text that consist of a single lowercase name; such a name is
// kept if the font has an icon for it. Stored icons (Spaces, document types, Tags) come from
// the built-in library's Go literals, so they are covered as well.
//
// The font maps names to icons with ligatures. Subsetting by text would keep every ligature
// whose letters remain, so the icons' ligature glyphs are selected explicitly and layout
// closure is disabled.
import fs from 'node:fs';
import path from 'node:path';
import * as hb from 'harfbuzzjs';
import fontverter from 'fontverter';

const sourceFontPath = 'ui/uix/web/fonts/material-symbols-outlined.woff2';
const targetFontPath = 'ui/uix/web/assets/fonts/material-symbols-outlined.woff2';
const sourceRoots = ['action', 'common', 'core', 'model', 'pluginx', 'server', 'ui'];
const skippedDirs = new Set(['node_modules', 'vendor', '.git']);
const iconNameChars = 'abcdefghijklmnopqrstuvwxyz0123456789_';
const isIconNameCandidate = value => /^[a-z0-9][a-z0-9_]{1,62}$/.test(value);

// HB_SUBSET_FLAGS_NO_LAYOUT_CLOSURE, see hb-subset.h
const noLayoutClosureFlag = 0x200;
// HB_MEMORY_MODE_WRITABLE, see hb-blob.h
const writableMemoryMode = 2;

function sourceFiles(dir) {
	return fs.readdirSync(dir, { withFileTypes: true }).flatMap(entry => {
		const entryPath = path.join(dir, entry.name);
		if (entry.isDirectory()) {
			return skippedDirs.has(entry.name) ? [] : sourceFiles(entryPath);
		}
		const isSource = /\.(go|gohtml|js)$/.test(entry.name) && !entry.name.endsWith('_test.go');
		return isSource ? [entryPath] : [];
	});
}

function delimitTemplateActions(content) {
	const parts = [];
	let offset = 0;
	while (offset < content.length) {
		const start = content.indexOf('{{', offset);
		if (start === -1) break;
		const end = content.indexOf('}}', start + 2);
		if (end === -1) break;
		parts.push(content.slice(offset, start), '<>');
		offset = end + 2;
	}
	parts.push(content.slice(offset));
	return parts.join('');
}

function iconNameCandidates() {
	const candidates = new Set();
	const add = value => {
		const trimmed = value.trim();
		if (isIconNameCandidate(trimmed)) candidates.add(trimmed);
	};
	for (const file of sourceRoots.flatMap(sourceFiles)) {
		const content = fs.readFileSync(file, 'utf8');
		for (const match of content.matchAll(/"([^"\n]*)"|'([^'\n]*)'|`([^`]*)`/g)) {
			add(match[1] ?? match[2] ?? match[3]);
		}
		if (file.endsWith('.gohtml')) {
			// template actions delimit text too, as in {{ if .IsOpen }}expand_less{{ end }}
			const text = delimitTemplateActions(content);
			for (const match of text.matchAll(/>([^<>]*)</g)) {
				add(match[1]);
			}
		}
	}
	return candidates;
}

function shape(font, text) {
	const buffer = new hb.Buffer();
	buffer.addText(text);
	buffer.guessSegmentProperties();
	hb.shape(font, buffer);
	return buffer.getGlyphInfos().map(info => info.codepoint);
}

// Returns the icon's ligature glyph, or -1 if the name is not an icon.
function iconGlyph(font, name) {
	const glyphs = shape(font, name);
	return glyphs.length === 1 ? glyphs[0] : -1;
}

async function subset(sfnt, glyphIDs) {
	const wasm = await WebAssembly.instantiate(
		fs.readFileSync('node_modules/harfbuzzjs/dist/harfbuzz-subset.wasm'),
	);
	const exports = wasm.instance.exports;
	const heap = () => new Uint8Array(exports.memory.buffer);

	const fontPtr = exports.malloc(sfnt.byteLength);
	heap().set(sfnt, fontPtr);
	const blob = exports.hb_blob_create(fontPtr, sfnt.byteLength, writableMemoryMode, 0, 0);
	const face = exports.hb_face_create(blob, 0);
	exports.hb_blob_destroy(blob);

	const input = exports.hb_subset_input_create_or_fail();
	if (input === 0) throw new Error('hb_subset_input_create_or_fail failed');
	const unicodes = exports.hb_subset_input_unicode_set(input);
	for (const char of iconNameChars) exports.hb_set_add(unicodes, char.codePointAt(0));
	const glyphs = exports.hb_subset_input_glyph_set(input);
	for (const glyphID of glyphIDs) exports.hb_set_add(glyphs, glyphID);
	exports.hb_subset_input_set_flags(
		input,
		exports.hb_subset_input_get_flags(input) | noLayoutClosureFlag,
	);

	const subsetFace = exports.hb_subset_or_fail(face, input);
	exports.hb_subset_input_destroy(input);
	if (subsetFace === 0) throw new Error('hb_subset_or_fail failed');
	const resultBlob = exports.hb_face_reference_blob(subsetFace);
	const offset = exports.hb_blob_get_data(resultBlob, 0);
	const length = exports.hb_blob_get_length(resultBlob);
	const result = Buffer.from(heap().slice(offset, offset + length));

	exports.hb_blob_destroy(resultBlob);
	exports.hb_face_destroy(subsetFace);
	exports.hb_face_destroy(face);
	exports.free(fontPtr);
	return result;
}

function loadFont(sfnt) {
	return new hb.Font(new hb.Face(new hb.Blob(sfnt), 0));
}

const sourceSfnt = await fontverter.convert(fs.readFileSync(sourceFontPath), 'sfnt');
const source = loadFont(sourceSfnt);
const icons = new Map();
for (const name of iconNameCandidates()) {
	const glyph = iconGlyph(source, name);
	if (glyph !== -1) icons.set(name, glyph);
}
const subsetSfnt = await subset(sourceSfnt, new Set([0, ...icons.values()]));

// Check every icon still renders the same outline; ligatures spanning several lookups would
// otherwise break silently.
const target = loadFont(subsetSfnt);
for (const [name, glyph] of icons) {
	const subsetGlyph = iconGlyph(target, name);
	if (subsetGlyph === -1 ||
		target.glyphToPath(subsetGlyph) !== source.glyphToPath(glyph)) {
		throw new Error(`icon ${name} does not render correctly in the subset font`);
	}
}

const subsetWoff2 = await fontverter.convert(subsetSfnt, 'woff2');
fs.writeFileSync(targetFontPath, subsetWoff2);
console.log(
	`kept ${icons.size} icons; ${targetFontPath}: ${subsetWoff2.byteLength} bytes ` +
	`(full font: ${fs.statSync(sourceFontPath).size} bytes)`,
);
