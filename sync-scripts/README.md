This directory holds scripts which can be executed by Syncers to get a list of images that need to be loaded into a registry.
Each script here must be an executable binary or bash file, and may be as simple or complex as need. However, you MUST ensure the output is a list of images seperated by new lines


For example, a valid output looks like the following 

harrisonwaffel/image1:latest
harrisonwaffel/image2:latest
harrisonwaffel/image3:v1.3.2

Any unexpected output WILL prevent the syncing process from occurring. 

