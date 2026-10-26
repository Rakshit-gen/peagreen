#!/bin/sh
# Downloads the public test ROMs the tests run against into testdata/roms.
# None of them are checked in: they're other people's work, under their own
# licenses. Every download is checked against a pinned SHA-256.
set -eu

dir=$(cd "$(dirname "$0")/.." && pwd)/testdata/roms
mkdir -p "$dir"
cd "$dir"

fetch() { # url file sha256
	if [ -f "$2" ] && echo "$3  $2" | shasum -a 256 -c - >/dev/null 2>&1; then
		return
	fi
	echo "fetching $2"
	curl -fsSL -o "$2.part" "$1"
	if ! echo "$3  $2.part" | shasum -a 256 -c - >/dev/null; then
		echo "$2: checksum mismatch" >&2
		rm -f "$2.part"
		exit 1
	fi
	mv "$2.part" "$2"
}

# blargg's test ROMs
fetch https://github.com/retrio/gb-test-roms/archive/c240dd7d700e5c0b00a7bbba52b53e4ee67b5f15.zip \
	blargg.zip 051264df1cca5a5fdce298289f53c960a6fab83e9d052c89d4091f298f757335
# dmg-acid2 and its reference image
fetch https://github.com/mattcurrie/dmg-acid2/releases/download/v1.0/dmg-acid2.gb \
	dmg-acid2.gb 464e14b7d42e7feea0b7ede42be7071dc88913f75b9ffa444299424b63d1dff1
fetch https://raw.githubusercontent.com/mattcurrie/dmg-acid2/8a98ce731f96dde032ffb22ec36dc985d78fdb18/img/reference-dmg.png \
	dmg-acid2.png ca966d50895c7efef05838590d148c2cbfd7fba57dab986f25b35b4da71abb57
# Gekkio's mooneye test suite, prebuilt
fetch https://gekkio.fi/files/mooneye-test-suite/mts-20240926-1737-443f6e1/mts-20240926-1737-443f6e1.zip \
	mooneye.zip 5cf50314cd3d42ec8f423b0a71b9ab6dbfad2386d8d88f03687af1b5b5baba71
# the picture mooneye's manual-only sprite_priority should show
fetch https://raw.githubusercontent.com/Gekkio/mooneye-test-suite/31510e12eea6286d36eea060a6adde755e1067aa/manual-only/sprite_priority-expected.png \
	sprite_priority.png e08708fd0e5fc7df8650314707ec0ce0c929b969f2e19b6f2860c1e82828e9a0

[ -d blargg ] || { unzip -q blargg.zip && mv gb-test-roms-* blargg; }
[ -d mooneye ] || { unzip -q mooneye.zip && mv mts-* mooneye; }
echo "test roms in $dir"
