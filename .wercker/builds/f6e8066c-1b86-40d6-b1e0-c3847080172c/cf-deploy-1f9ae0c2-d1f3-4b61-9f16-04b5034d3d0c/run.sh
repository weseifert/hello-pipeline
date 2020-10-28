#! /bin/bash
export PATH=`pwd`:$WERCKER_STEP_ROOT:$PATH

eval "$WERCKER_STEP_ROOT/cf-push-step $@";
