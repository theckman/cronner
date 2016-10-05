#!/bin/bash

echo "stdout"
(>&2 echo "stderr")
(>&2 echo "stderr")
echo "stdout"
(>&2 echo "stderr")
echo "stdout"
echo "stdout"
(>&2 echo "stderr")
