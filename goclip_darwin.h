#include "Cocoa/Cocoa.h"

NSPasteboard *cocoaPbFactory();
int cocoaPbChangeCount(NSPasteboard *pb);
void pasteWriteAddText(char* data, int len);
void pasteWrite(NSPasteboard* pb);

typedef struct ClipboardInformation {
	int typeInt;
	int formatTypeInt;
	int count;
} ClipboardInformation;

typedef struct ClipboardData {
	Byte* data;
	int dataLength;
} ClipboardData;

typedef struct ClipboardTypeFilter {
    bool text;
    bool image;
    bool files;
} ClipboardTypeFilter;

void readClipboard(NSPasteboard *pb, ClipboardData* cbData, ClipboardInformation *cbInfo, ClipboardTypeFilter *filter);
void readInformation(NSPasteboard *pb, ClipboardInformation *cbInfo);

const char* nsstring2cstring(NSString *str);
