#!/bin/bash

failures=0
trap 'failures=$((failures+1))' ERR

# Installing litmus
if [ "$INSTALL_LITMUS" == "true" ];then
    #running install-litmus go binary to install litmus
    ./install-litmus
fi

## catching failure and uninstalling litmus
if ((failures != 0)); then
  if [ "$UNINSTALL_LITMUS" == "true" ];then
  #running uninstall-litmus go binary 
  ./uninstall-litmus
  fi
  echo "$failures failures found"
  exit 1
fi
#execute desired chaosexperiment
if [ ! -z "$EXPERIMENT_NAME" ];then
    #running experiment go binary 
    ./$EXPERIMENT_NAME
else
    echo "No experiment to run. Please setup EXPERIMENT_NAME env to run an experiment"
    if [ "$UNINSTALL_LITMUS" == "true" ];then
    ./uninstall-litmus
    fi
    exit 1
fi

## catching failure and uninstalling litmus
if ((failures != 0)); then
  if [ "$UNINSTALL_LITMUS" == "true" ];then
  #running uninstall-litmus go binary 
  ./uninstall-litmus
  fi
  echo "$failures failures found"
  exit 1
fi

# Uninstall litmus
if [ "$UNINSTALL_LITMUS" == "true" ];then
    #running uninstall-litmus go binary 
    ./uninstall-litmus
fi
