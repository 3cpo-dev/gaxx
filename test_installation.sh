#!/bin/bash
echo "Testing gaxx installation..."
which gaxx
gaxx --help | head -3
echo "✅ Success! You can now use: gaxx spawn, gaxx ls, gaxx run, etc."
