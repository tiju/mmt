# MMT

**M**edia **M**anagement **T**ool

Or, what to do if your desk looks like this:

![](https://i.imgur.com/qmgLaxg.jpg)

## Backstory:

I've been using an assortment of scripts over the years to manage media from my different action cameras and drones, it's clear a centralized and unified solution is needed.

This tool draws inspiration from my [dji-utils/offload.sh](https://github.com/KonradIT/djiutils/blob/master/offload.sh) script as well as the popular [gopro-linux tool](https://github.com/KonradIT/gopro-linux/blob/master/gopro#L262) and @deviantollam's [dohpro](https://github.com/deviantollam/dohpro)

Right now the script supports these cameras:

-   GoPro: Pretty much all of them

To be supported:

-   DJI: Tested with Osmo Pocket, Spark and Mavic Air 2, but should work on Osmo action and other drones as well
-   Insta360: X2
-   Android: photos and videos recorded with OnePlus 7T, but possibly most Android phones

Feel free to PR!

I plan have the tool read a directory, use a config file and act accordingly to offload media from any type of drive

## Installing:

Download from the releases tab, a github action will run for every push.

## Running:

-   import
    -   `--input`: A directory pointing to your SD card, or MTP connection, on Windows it would be a letter (eg: `E:\`)
    -   `--output`: Destination folder, a hard drive, etc...
    -   `--name`: Project name, eg: `Paragliding Weekend Winter 2021`
-   update
    -   `--input`: A directory pointing to your SD card

## To-do:

-   [ ] **HiLight parsing**: I've found that the best way to see which clip I will use later on is to put some tags at the end of it (press mode button on GoPro, or shout "Oh Shit", or use the app/pebble app). Then when I run a script that prints the number of hilight tags during the last 30 seconds of each video. That lets me know the clips are important. This tool should let you label each tag count (eg: --tag-labels="good,great,important") for each hilight count.
-   [ ] **Sort by location**: Should be on root, so:

    ```
    - Mexico City, Mexico:
       - 2020-01-02:
    	     ...
    - New York, NY, United States:
       - 2017-07-01:
    	     ...
    - Madrid, Spain:
       - 2020-09-02:
    	     ...

    ```

    To get location info: GoPro ([GPMF](https://github.com/stilldavid/gopro-utils)) DJI (SRT file) Insta360 (???)

-   [ ] **Date range**: Import from only certain dates (allow for: `today`, `yesterday` and `week`, `--date-start` and `--date-end`)
-   [ ] **Extract info from each clip**: Eg: km travelled, altitude changes, number of faces, shouts, etc...
-   [ ] **Merging chapters**: GoPro only, merge chapters from separate files
-   [ ] **Generate GIF for burst photos**: Move each burst sequence to a separate folder and make a GIF
-   [ ] **Merge timelapse photos**: Using ffmpeg
-   [ ] **Generate DNG from GPR**: Using [gpr tool](https://github.com/gopro/gpr)
-   [x] **Proxy file support**
-   [ ] **H265 to H264 conversion**: Using ffmpeg
-   [x] **Update camera firmware?** (Done: GoPro, pending: Insta360)
-   [ ] **Use goroutines**
-   [ ] **Tests**
-   [ ] **Import media from GoPro's Webcam mode (USB Ethernet)**
