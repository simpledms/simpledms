import * as esbuild from "esbuild";

const outdir = "ui/uix/web/assets/vendor";

// run with `node ./install.deps.mjs`
esbuild
	.build({
		bundle: true,
		format: "esm",
		// same path used below
		outdir: outdir,
		outbase: ".", // necessary to preserve path
		platform: "browser",
		splitting: true,
		minify: true,
		sourcemap: true,
		entryPoints: [
			"pdfjs-dist",
			"pdfjs-dist/build/pdf.worker.mjs",
			"htmx.org",
			"idiomorph/dist/idiomorph-ext.esm.js",
			"@uppy/core",
			"@uppy/xhr-upload",
			"@uppy/dashboard",
			//"@uppy/url", // needs companion
			"@uppy/webcam",
			"@uppy/audio",
			"@uppy/screen-capture",
			"@uppy/core/dist/style.min.css",
			"@uppy/dashboard/dist/style.min.css",
			//"@uppy/url/dist/style.min.css",
			"@uppy/webcam/dist/style.min.css",
			"@uppy/audio/dist/style.min.css",
			"@uppy/screen-capture/dist/style.min.css",
			"@uppy/locales/lib/de_DE",
			"@uppy/locales/lib/fr_FR",
			"@uppy/locales/lib/it_IT",
		],
	})
	.then((res) => {
		// res doesn't hold any information...
		console.log("esbuild finished");
	});

esbuild
	.build({
		bundle: true,
		format: "iife",
		outdir: outdir,
		outbase: ".", // necessary to preserve path
		platform: "browser",
		minify: true,
		sourcemap: true,
		entryPoints: [
			"htmx.org/dist/ext/debug.js",
		],
	})
	.then((res) => {
		// res doesn't hold any information...
		console.log("esbuild finished");
	});
