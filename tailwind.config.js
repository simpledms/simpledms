/** @type {import('tailwindcss').Config} */
const defaultTheme = require('tailwindcss/defaultTheme');
const plugin = require('tailwindcss/plugin');

module.exports = {
	content: [
		//'../../ui/widget/**/*.gohtml',
		//'../../ui/widget/**/*.go',
		'./ui/widget/**/*.gohtml',
		'./ui/widget/**/*.go'
	],
	theme: {
		// based on https://github.com/material-foundation/material-tokens/tree/main/css
		// conversion from px to rem is based on root font-size of 16px
		// https://m3.material.io/styles
		screens: {
			// from material
			// default is compact
			'md': '600px', // medium
			'exp': '840px', // expanded // TODO ex or exp?
			'lg': '1200px', // large
			'xl': '1600px', // extra-large
		},
		fontSize: {
			/*
			'label-sm':'0.6875rem', // 11px
			'label-md':'0.75rem',
			'label-lg':'0.875rem',
			'body-sm':'0.75rem',
			'body-md':'1.25rem',
			'body-lg':'1.5rem',
			'title-sm':'1.25rem',
			'title-md':'1.5rem',
			'title-lg':'1.75rem',
			'headline-sm':'2rem',
			'headline-md':'2.25rem',
			'headline-lg':'2.5rem',
			'display-sm':'2.75rem',
			'display-md':'3.25rem',
			'display-lg':'4rem',
			 */
		},
		borderRadius: {
			// TODO guessed
			// DEFAULT: '',
			'none': '0',
			'xs': '0.25rem',
			'sm': '0.5rem',
			'md': '0.75rem',
			'lg': '1rem',
			'xl': '1.75rem',
			'2xl': '2.25rem',
			'full': '9999px' // TODO good value? 100% doesn't work well in many cases
		},
		// taken DP from elevation spec: https://m3.material.io/styles/elevation/tokens
		// lighter ones (above 1) for dark mode
		brightness: {
			75: '.88',
			80: '.92',
			85: '.94',
			90: '.97',
			95: '.99',
			100: '1',
			105: '1.01',
			110: '1.03',
			115: '1.06',
			120: '1.08',
			125: '1.12',
		},
		colors: {
			// doesn't work when switching between light and dark mode
			// 'white': '#ffffff',
			// 'black': '#000000',
			'transparent': 'transparent',

			// colors are defined in tailwind.css, alpha-value is necessary for opacity to work
			'primary': 'rgb(var(--md-sys-color-primary) / <alpha-value>)',
			'surface-tint': 'rgb(var(--md-sys-color-surface-tint) / <alpha-value>)',
			'on-primary': 'rgb(var(--md-sys-color-on-primary) / <alpha-value>)',
			'primary-container': 'rgb(var(--md-sys-color-primary-container) / <alpha-value>)',
			'on-primary-container': 'rgb(var(--md-sys-color-on-primary-container) / <alpha-value>)',
			'secondary': 'rgb(var(--md-sys-color-secondary) / <alpha-value>)',
			'on-secondary': 'rgb(var(--md-sys-color-on-secondary) / <alpha-value>)',
			'secondary-container': 'rgb(var(--md-sys-color-secondary-container) / <alpha-value>)',
			'on-secondary-container': 'rgb(var(--md-sys-color-on-secondary-container) / <alpha-value>)',
			'tertiary': 'rgb(var(--md-sys-color-tertiary) / <alpha-value>)',
			'on-tertiary': 'rgb(var(--md-sys-color-on-tertiary) / <alpha-value>)',
			'tertiary-container':  'rgb(var(--md-sys-color-tertiary-container) / <alpha-value>)',
			'on-tertiary-container': 'rgb(var(--md-sys-color-on-tertiary-container) / <alpha-value>)',
			'error': 'rgb(var(--md-sys-color-error) / <alpha-value>)',
			'on-error': 'rgb(var(--md-sys-color-on-error) / <alpha-value>)',
			'error-container': 'rgb(var(--md-sys-color-error-container) / <alpha-value>)',
			'on-error-container': 'rgb(var(--md-sys-color-on-error-container) / <alpha-value>)',
			'background': 'rgb(var(--md-sys-color-background) / <alpha-value>)',
			'on-background': 'rgb(var(--md-sys-color-on-background) / <alpha-value>)',
			'surface': 'rgb(var(--md-sys-color-surface) / <alpha-value>)',
			'on-surface': 'rgb(var(--md-sys-color-on-surface) / <alpha-value>)',
			'surface-variant': 'rgb(var(--md-sys-color-surface-variant) / <alpha-value>)',
			'on-surface-variant': 'rgb(var(--md-sys-color-on-surface-variant) / <alpha-value>)',
			'outline': 'rgb(var(--md-sys-color-outline) / <alpha-value>)',
			'outline-variant': 'rgb(var(--md-sys-color-outline-variant) / <alpha-value>)',
			'shadow': 'rgb(var(--md-sys-color-shadow) / <alpha-value>)',
			'scrim': 'rgb(var(--md-sys-color-scrim) / <alpha-value>)',
			'inverse-surface': 'rgb(var(--md-sys-color-inverse-surface) / <alpha-value>)',
			'inverse-on-surface': 'rgb(var(--md-sys-color-inverse-on-surface) / <alpha-value>)',
			'inverse-primary': 'rgb(var(--md-sys-color-inverse-primary) / <alpha-value>)',
			'primary-fixed': 'rgb(var(--md-sys-color-primary-fixed) / <alpha-value>)',
			'on-primary-fixed': 'rgb(var(--md-sys-color-on-primary-fixed) / <alpha-value>)',
			'primary-fixed-dim': 'rgb(var(--md-sys-color-primary-fixed-dim) / <alpha-value>)',
			'on-primary-fixed-variant': 'rgb(var(--md-sys-color-on-primary-fixed-variant) / <alpha-value>)',
			'secondary-fixed': 'rgb(var(--md-sys-color-secondary-fixed) / <alpha-value>)',
			'on-secondary-fixed': 'rgb(var(--md-sys-color-on-secondary-fixed) / <alpha-value>)',
			'secondary-fixed-dim': 'rgb(var(--md-sys-color-secondary-fixed-dim) / <alpha-value>)',
			'on-secondary-fixed-variant': 'rgb(var(--md-sys-color-on-secondary-fixed-variant) / <alpha-value>)',
			'tertiary-fixed': 'rgb(var(--md-sys-color-tertiary-fixed) / <alpha-value>)',
			'on-tertiary-fixed': 'rgb(var(--md-sys-color-on-tertiary-fixed) / <alpha-value>)',
			'tertiary-fixed-dim': 'rgb(var(--md-sys-color-tertiary-fixed-dim) / <alpha-value>)',
			'on-tertiary-fixed-variant':'rgb(var(--md-sys-color-on-tertiary-fixed-variant) / <alpha-value>)',
			'surface-dim': 'rgb(var(--md-sys-color-surface-dim) / <alpha-value>)',
			'surface-bright': 'rgb(var(--md-sys-color-surface-bright) / <alpha-value>)',
			'surface-container-lowest': 'rgb(var(--md-sys-color-surface-container-lowest) / <alpha-value>)',
			'surface-container-low': 'rgb(var(--md-sys-color-surface-container-low) / <alpha-value>)',
			'surface-container': 'rgb(var(--md-sys-color-surface-container) / <alpha-value>)',
			'surface-container-high': 'rgb(var(--md-sys-color-surface-container-high) / <alpha-value>)',
			'surface-container-highest': 'rgb(var(--md-sys-color-surface-container-highest) / <alpha-value>)',
			'beige-color': 'rgb(var(--md-extended-color-beige-color) / <alpha-value>)',
			'beige-on-color': 'rgb(var(--md-extended-color-beige-on-color) / <alpha-value>)',
			'beige-color-container': 'rgb(var(--md-extended-color-beige-color-container) / <alpha-value>)',
			'beige-on-color-container': 'rgb(var(--md-extended-color-beige-on-color-container) / <alpha-value>)',
			'aliceblue-color': 'rgb(var(--md-extended-color-aliceblue-color) / <alpha-value>)',
			'aliceblue-on-color': 'rgb(var(--md-extended-color-aliceblue-on-color) / <alpha-value>)',
			'aliceblue-color-container': 'rgb(var(--md-extended-color-aliceblue-color-container) / <alpha-value>)',
			'aliceblue-on-color-container': 'rgb(var(--md-extended-color-aliceblue-on-color-container) / <alpha-value>)',
		},
		extend: {
			colors: {
				// creates complete color palettes
			},
			opacity: {
				'8': '0.08',
				'12': '0.12',
				'88': '0.88',
				'92': '0.92',
			},
			keyframes: {
				'indeterminate-linear': {
					'0%': { transform: 'translateX(-100%)' },
					'100%': { transform: 'translateX(100%)' }
				}
			},
			animation: {
				'indeterminate-linear-progress': 'indeterminate-linear 1.5s infinite ease-in-out'
			},
            fontFamily: {
                sans: ['"Noto Sans"', ...defaultTheme.fontFamily.sans]
            },
		}
	},
	plugins: [
		/*
		require('@tailwindcss/forms')({
			// don't generate classes
			strategy: "base",
		}),
		 */
		require('@tailwindcss/typography'),
		require('@tailwindcss/container-queries'),
		plugin(function({ addVariant }) {
			// TODO add focus-within? not sure if it can have side effects
			addVariant('fa', ['&:focus', '&:active'])
			addVariant('hfa', ['&:hover', '&:focus', '&:active'])
			addVariant('group-fa', ['&:group-focus', '&:group-active'])
			addVariant('group-hfa', ['&:group-hover', '&:group-focus', '&:group-active'])
		}),
		plugin(function({ matchUtilities, theme }) {
			matchUtilities({
				'state-layer': (value) => ({
					//'display': 'inherit',
					'height': 'inherit',
					'width': 'inherit',
					'border-radius': 'inherit',
					'overflow': 'auto', // TODO good?
					// not sure if good idea to have normal and group together, thus group always applied
					'&:hover, .group:hover &': {
						// TODO maybe there is a nicer way? just important that `var(--xyz)` is used
						// 		in final version to have it replaceable
						backgroundColor: value.replace(/<alpha-value>/g, '0.08'),
					},
					// .group:focus-within is not tested as of 18.03.25
					'&:focus, &:focus-within, .group:focus &, .group:focus-within &': {
						backgroundColor: value.replace(/<alpha-value>/g, '0.10'),
					},
					'&:active, .group:active &': {
						backgroundColor: value.replace(/<alpha-value>/g, '0.10'),
					},
					// TODO drag
				}),
			}, {
				// type: ['color'], // value is corrupted if used
				values: theme('colors'), // TODO flatten?
			})
		})
	],
}
