> [!IMPORTANT]
> **This project is discontinued**: Git works better. [Here's how to set it up!](https://github.com/claabs/watch-dogs-legion-version-manager/blob/master/GitAlternative.md)

# Watch Dogs Legion Version Manager

Easily downgrade and upgrade your Watch Dogs: Legion game version in Ubisoft Connect.

## Version Dates

| Date       | Version  | Patch Notes                                                                                                                                                                                |
|------------|----------|--------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| 2020-10-26 | 1.0.00*  | Preload                                                                                                                                                                                    |
| 2020-10-28 | 1.0.10*  | Day 1 Patch                                                                                                                                                                                |
| 2020-10-30 | 1.1.00*  | [Link](https://web.archive.org/web/20220331011627/https://forums.ubisoft.com/showthread.php/2285126-*COMPLETE*-Maintenance-for-Hotfix-Patch-October-28-(Xbox-amp-PS4)-amp-October-30-(PC)) |
| 2020-11-06 | 1.2.00   | [Link](https://web.archive.org/web/20220327230309/https://forums.ubisoft.com/showthread.php/2289326-Bug-Fixes-for-PlayStation-4-Xbox-One-and-Stadia-TU-2-0)                                |
| 2020-11-12 | 1.2.10** | [Link](https://web.archive.org/web/20220331061050/https://forums.ubisoft.com/showthread.php/2291783-TU2-10-Bug-fixes)                                                                      |
| 2020-11-26 | 1.2.20   | [Link](https://web.archive.org/web/20220331111258/https://forums.ubisoft.com/showthread.php/2297075-Patch-Notes-TU-2-20)                                                                   |
| 2020-12-02 | 1.2.30   | [Link](https://web.archive.org/web/20220331042232/https://forums.ubisoft.com/showthread.php/2299330-TU-2-30-Patch-Notes)                                                                   |
| 2020-12-10 | 1.2.40   | [Link](https://web.archive.org/web/20220331125617/https://forums.ubisoft.com/showthread.php/2302093-Title-Update-2-40-Patch-Notes)                                                         |
| 2021-01-27 | 1.3.00   | [Link](https://web.archive.org/web/20220331081614/https://forums.ubisoft.com/showthread.php/2315110-Title-Update-3-0-Patch-Notes)                                                          |
| 2021-02-24 | 1.3.10** | [Link](https://web.archive.org/web/20220331021548/https://forums.ubisoft.com/showthread.php/2323279-TU-3-1-Console-Update)                                                                 |
| 2021-03-08 | 1.3.20** | [Link](https://web.archive.org/web/20220331072050/https://forums.ubisoft.com/showthread.php/2325930-Title-Update-3-20-Patch-Notes)                                                         |
| 2021-03-16 | 1.3.21** | [Link](https://web.archive.org/web/20220331120400/https://forums.ubisoft.com/showthread.php/2327539-Hotfix-3-21-Patch-Notes)                                                               |
| 2021-03-18 | 1.3.22   | [Link](https://web.archive.org/web/20220331104540/https://forums.ubisoft.com/showthread.php/2327849-Title-Update-3-22-and-Online-Mode-for-PC-Patch-Notes)                                  |
| 2021-03-22 | 1.3.25   | [Link](https://web.archive.org/web/20220330192640/https://forums.ubisoft.com/showthread.php/2330961-Title-Update-3-25-Patch-Notes)                                                         |
| 2021-04-12 | 1.3.30   | [Link](https://web.archive.org/web/20220331144656/https://forums.ubisoft.com/showthread.php/2336678-Title-Update-3-30)                                                                     |
| 2021-05-04 | 1.4.00   | [Link](https://web.archive.org/web/20210723162441/https://forums.ubisoft.com/showthread.php/2340084-Title-Update-4-0-%E2%80%93-Patch-Notes)                                                |
| 2021-05-19 | 1.4.02   | [Link](https://web.archive.org/web/20231030021458/https://nitter.net/watchdogsgame/status/1395002581253070850)                                                                             |
| 2021-06-01 | 1.4.50   | [Link](https://web.archive.org/web/20220124190644/https://forums.ubisoft.com/showthread.php/2346244-Title-Update-4-5-%E2%80%93-Patch-Notes)                                                |
| 2021-07-02 | 1.5.00   | [Link](https://web.archive.org/web/20220320132939/https://forums.ubisoft.com/showthread.php/2352504-Title-Update-5-0-%E2%80%93-Patch-Notes)                                                |
| 2021-08-24 | 1.5.50   | [Link](https://web.archive.org/web/20220127054640/https://discussions.ubisoft.com/topic/104317/title-update-5-5-patch-notes)                                                               |
| 2021-09-02 | 1.5.51   | [Link](https://web.archive.org/web/20210902150340/https://discussions.ubisoft.com/topic/106738/tu-5-51-patch-notes?lang=en-US)                                                             |
| 2021-09-14 | 1.5.60   | [Link](https://web.archive.org/web/20230306221621/https://discussions.ubisoft.com/topic/108299/title-update-5-6-patch-notes?lang=en-US)                                                    |
| 2023-10-31 | 1.6.30   | [Link](https://steamdb.info/depot/2239551/history/)                                                                                                                                        |

\**Version had no official number, so an estimated one is used*

\*\**Console-only update*

## Configuration

* **currentgameversion:** The current version state of the game. Feel free to change it if desynced. (Default: `<latest version>`)
* **cachepath:** The location of the cached version files so you don't need to redownload files all the time. It is recommended to keep this on the same disk as your game to greatly speed up transfer times. (Default: `%PROGRAMFILES(X86)%\Ubisoft\Ubisoft Game Launcher\games\Watch Dogs Legion Version Cache`)
* **gamepath:** The location of your game install (Default: `%PROGRAMFILES(X86)%\Ubisoft\Ubisoft Game Launcher\games\Watch Dogs Legion`)
* **savepath:** The location of your game save files (Default: `%PROGRAMFILES(X86)%\Ubisoft\Ubisoft Game Launcher\savegames\<uplay-user-id>\3353`)
* **fastprocessing:** Process all the files in parallel (Default: `false`)
* **fastdownload:** Use Accept-Ranges partial file download to speed up individual file download (Default: `false`)

## Troubleshooting

If you see that the version changer is missing files, or has produced empty files due to a cancelled download, the following steps can reset the file setup:

1. Set the `currentgameversion` in config.yml to the latest game version
1. Clear your cache file folder
1. Verify and repair your game files in Ubisoft Connect

If you're having trouble with downloads failing, try disabling `fastprocessing` and/or `fastdownload` in the config.

## To Do

* Progress bar on file moves (between drives)
* Remove slow download mode
* Add CRC checksums to verify file version

## Development

### Server hosting (UNIX)

Hosting files on cloud providers can be **very expensive** due to data transfer costs. Even just a few downloads a month is about $100 on AWS. The best approach for me was to just host it on my home server; just make sure you have enough storage and decent upload speeds.

1. Be sure to archive files for the game as the patches roll out. Use Windows backup, or setup [Gitea](https://github.com/go-gitea/gitea) and commit files via [Git-LFS](https://git-lfs.github.com/)
1. Place the files in a folder, labeled with the version as the suffix (e.g. `file.txt.1.0.0`)
1. Create a `versions.txt` with each version on a new line
1. Create a `files.txt` with all the files to track (with path and without version extension)
1. Generate CRCs for all the files in a SFV file:
   * `find * -type f \( ! -iname "*.txt" \) -exec sh -c 'echo "$1" $(cksum {} | cut -d " " -f 1 -)' sh {} \; > files.sfv`
1. Run a static file hosting server on the folder containing the files
   * I use [static-web-server](https://github.com/joseluisq/static-web-server) because it supports partial downloads

### Implementation notes

#### Downgrade steps

1. Have Ubisoft Connect open in online mode
1. Disable auto updates in Ubisoft Connect
1. Get list of archive files by date for desired version
1. Rename files to be replaced with latest version number
1. Download archive files and rename to replace former latest files
1. Save file noting the current installed version
1. Back up and delete latest version saves
1. Launch the game
1. Exit the game and switch Ubisoft Connect to offline mode
1. Add any desired practice save files
1. Launch the game

#### How to check one file

1. Get the actual current version of the file
   * Get latest version number from remote
   * Remember actual current version number
1. Get actual desired version of the file
   * Get latest version number from remote
   * Remember actual desired version number
1. If actual current version and actual desired version are different:
   * Cache current file with actual current version number
   * Obtain actual desired version number from cache or remote and copy it to game location
1. Update current version number

*How do we undo a broken downgrade?*
