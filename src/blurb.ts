import qrcode from 'qrcode-generator';
import sanitizeHtml from 'sanitize-html';

export default class BlurbStuff {

    static write_at(arr: Uint8Array, offset: number, value: number) {
        while (value > 0) {
            arr[offset] = value & 0xFF;
            value >>= 8;
            offset++;
        }
    }

    static get_qr_code(url: string): string {
        const qr = qrcode(0, 'L');
        qr.addData(url, "Byte");
        qr.make();

        const moduleCount = qr.getModuleCount();
        // BMP header is 62 bytes for our configuration
        const rowSize = Math.floor((moduleCount + 31) / 32) * 4; // Padded to 4 bytes
        const imageSize = rowSize * moduleCount;
        const fileSize = 62 + imageSize;

        // BMP header
        const header = new Uint8Array(62);
        header.fill(0);
        // BMP signature
        header[0] = 0x42; // 'B'
        header[1] = 0x4D; // 'M'
        // File size
        this.write_at(header, 2, fileSize);
        // Offset to pixel data
        header[10] = 62;
        // DIB header size
        header[14] = 40;
        // Width
        this.write_at(header, 18, moduleCount);
        // Height
        this.write_at(header, 22, moduleCount);
        // Color planes
        header[26] = 1;
        // Bits per pixel
        header[28] = 1;
        // Image size
        this.write_at(header, 34, imageSize);
        // Resolution (18 DPI = 709 pixels/meter)
        this.write_at(header, 38, 709);
        this.write_at(header, 42, 709);
        // Color palette entries
        header[46] = 2;
        // Important colors
        header[50] = 2;
        // Color palette (black and white)
        this.write_at(header, 54, 0xFFFFFF); // White
        this.write_at(header, 58, 0x000000); // Black

        // Pixel data
        const pixels = new Uint8Array(imageSize);
        for (let y = 0; y < moduleCount; y++) {
            for (let x = 0; x < moduleCount; x++) {
                // BMP is bottom-up, left to right
                const invY = moduleCount - 1 - y;
                const byteIndex = Math.floor(x / 8) + invY * rowSize;
                const bitIndex = 7 - (x % 8);
                if (qr.isDark(y, x)) {
                    pixels[byteIndex] |= (1 << bitIndex);
                }
            }
        }

        // Combine header and pixels
        const bmpData = new Uint8Array(fileSize);
        bmpData.set(header);
        bmpData.set(pixels, 62);

        const bmp = btoa(String.fromCharCode.apply(null, Array.from(bmpData)));
        return bmp;
    }


    static sanitize_user_supplied_html(s: string): string {
        const rgb_regex = /\s*rgb\(\s*\d+\s*,\s*\d+\s*,\s*\d+\s*\)/;
        const clean = sanitizeHtml(s, {
            allowedTags: ['p', 'strong', 'em', 'u', 'sub', 'sup', 's', "span"],
            allowedAttributes: {
                '*': ['style']
            },
            allowedStyles: {
                '*': {
                    'color': [rgb_regex],
                    'background-color': [rgb_regex]
                }
            },
            transformTags: {
                '*': (tagName, attribs) => {
                    if (attribs.style) {
                        const styles = attribs.style.split(';').filter(s => s.trim());
                        const cleanStyles = styles.filter(style => {
                            const [prop, val] = style.split(':').map(s => s.trim());
                            return ['color', 'background-color'].includes(prop) &&
                                   /^rgb\(\s*\d+\s*,\s*\d+\s*,\s*\d+\s*\)$/.test(val);
                        });
                        attribs.style = cleanStyles.join(';');
                    }
                    return {
                        tagName,
                        attribs
                    };
                }
            }
        });
        return clean;
    }
}
