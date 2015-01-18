#!/bin/sh


qemu-system-x86_64 -enable-kvm -m 2048 -cpu core2duo \
  -machine q35 \
  -usb -device usb-kbd -device usb-mouse \
  -device isa-applesmc,osk="ourhardworkbythesewordsguardedpleasedontsteal(c)AppleComputerInc" \
  -kernel ./chameleon_svn2360_boot \
  -smbios type=2 \
  -device ide-drive,bus=ide.2,drive=MacHDD \
  -drive id=MacHDD,if=none,file=./ignore/10.8/osx.img \
  -netdev user,id=hub0port0 \
  -device e1000-82545em,netdev=hub0port0,id=mac_vnet0 \
  -device ide-drive,bus=ide.0,drive=MacDVD \
  -drive id=MacDVD,if=none,snapshot=on,file=./ignore/10.8/10.8.5.iso \
  -monitor stdio
