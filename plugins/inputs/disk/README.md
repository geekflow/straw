# Disk Plugin
Note that used_percent is calculated by doing used / (used + free), not used / total, which is how the unix df command does it. See https://en.wikipedia.org/wiki/Df_(Unix) for more details.

### Metrics:

- disk
  - tags:
    - fstype (filesystem type)
    - device (device file)
    - path (mount point path)
    - mode (whether the mount is rw or ro)
  - fields:
    - free (integer, bytes)
    - total (integer, bytes)
    - used (integer, bytes)
    - used_percent (float, percent)
    - inodes_free (integer, files)
    - inodes_total (integer, files)
    - inodes_used (integer, files)
    
    
#### https://github.com/influxdata/telegraf/tree/master/plugins/inputs/disk
