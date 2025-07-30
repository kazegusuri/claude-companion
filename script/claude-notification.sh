#!/bin/bash

if ! read -t 0; then
    exit
fi

read -r input
echo $input >> /var/log/claude-notification.log