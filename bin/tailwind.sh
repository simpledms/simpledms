#!/bin/sh
#npx tailwindcss -i ui/uix/web/tailwind.css -o ui/uix/web/assets/tailwind.css --watch
npx --ignore-scripts @tailwindcss/cli -i ui/uix/web/tailwind.css -o ui/uix/web/assets/tailwind.css --watch
