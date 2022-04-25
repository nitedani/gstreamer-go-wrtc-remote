/* eslint-disable sonarjs/cognitive-complexity */
function extractSdp(sdpLine: string, pattern: RegExp) {
  const result = sdpLine.match(pattern);
  return result && result.length === 2 ? result[1] : null;
}
function findLineInRange(
  sdpLines: string | any[],
  startLine: number,
  endLine: number,
  prefix: any,
  substr?: string,
) {
  const realEndLine = endLine !== -1 ? endLine : sdpLines.length;
  for (let i = startLine; i < realEndLine; ++i) {
    if (
      sdpLines[i].indexOf(prefix) === 0 &&
      (!substr ||
        sdpLines[i].toLowerCase().indexOf(substr.toLowerCase()) !== -1)
    ) {
      return i;
    }
  }
  return null;
}
function findLine(sdpLines: any, prefix: string, substr?: string | undefined) {
  return findLineInRange(sdpLines, 0, -1, prefix, substr);
}
function getCodecPayloadType(sdpLine: string) {
  const pattern = new RegExp('a=rtpmap:(\\d+) \\w+\\/\\d+');
  const result = sdpLine.match(pattern);
  return result && result.length === 2 ? result[1] : null;
}

export function setOpusAttributes(
  sdp: string,
  params: {
    [x: string]: any;
    stereo?: any;
    maxaveragebitrate?: any;
    maxplaybackrate?: any;
    cbr?: any;
    useinbandfec?: any;
    usedtx?: any;
    maxptime?: any;
  },
) {
  params = params || {};

  const sdpLines = sdp.split('\r\n');

  // Opus
  const opusIndex = findLine(sdpLines, 'a=rtpmap', 'opus/48000');
  let opusPayload;
  if (opusIndex) {
    opusPayload = getCodecPayloadType(sdpLines[opusIndex]);
  }

  if (!opusPayload) {
    return sdp;
  }

  const opusFmtpLineIndex = findLine(
    sdpLines,
    'a=fmtp:' + opusPayload.toString(),
  );
  if (opusFmtpLineIndex === null) {
    return sdp;
  }

  let appendOpusNext = '';
  appendOpusNext +=
    '; stereo=' + (typeof params.stereo != 'undefined' ? params.stereo : '1');
  appendOpusNext +=
    '; sprop-stereo=' +
    (typeof params['sprop-stereo'] != 'undefined'
      ? params['sprop-stereo']
      : '1');

  if (typeof params.maxaveragebitrate != 'undefined') {
    appendOpusNext +=
      '; maxaveragebitrate=' + (params.maxaveragebitrate || 128 * 1024 * 8);
  }

  if (typeof params.maxplaybackrate != 'undefined') {
    appendOpusNext +=
      '; maxplaybackrate=' + (params.maxplaybackrate || 128 * 1024 * 8);
  }

  if (typeof params.cbr != 'undefined') {
    appendOpusNext +=
      '; cbr=' + (typeof params.cbr != 'undefined' ? params.cbr : '1');
  }

  if (typeof params.useinbandfec != 'undefined') {
    appendOpusNext += '; useinbandfec=' + params.useinbandfec;
  }

  if (typeof params.usedtx != 'undefined') {
    appendOpusNext += '; usedtx=' + params.usedtx;
  }

  if (typeof params.maxptime != 'undefined') {
    appendOpusNext += '\r\na=maxptime:' + params.maxptime;
  }

  sdpLines[opusFmtpLineIndex] =
    sdpLines[opusFmtpLineIndex].concat(appendOpusNext);

  sdp = sdpLines.join('\r\n');
  return sdp;
}

export function forceStereoAudio(sdp: string) {
  const sdpLines = sdp.split('\r\n');
  let fmtpLineIndex: any = null;
  for (const line of sdpLines) {
    if (line.search('opus/48000') !== -1) {
      const opusPayload = extractSdp(line, /:(\d+) opus\/48000/i);

      for (let i = 0; i < sdpLines.length; i++) {
        if (sdpLines[i].search('a=fmtp') !== -1) {
          const payload = extractSdp(sdpLines[i], /a=fmtp:(\d+)/);
          if (payload === opusPayload) {
            fmtpLineIndex = i;
            break;
          }
        }
      }

      break;
    }
  }

  if (fmtpLineIndex === null) return sdp;
  sdpLines[fmtpLineIndex] = sdpLines[fmtpLineIndex].concat(
    '; stereo=1; sprop-stereo=1',
  );
  sdp = sdpLines.join('\r\n');
  return sdp;
}

export function setMediaBitrates(sdp: any) {
  return setMediaBitrate(setMediaBitrate(sdp, 'video', 10000), 'audio', 128);
}

function setMediaBitrate(sdp: string, media: string, bitrate: string | number) {
  const lines = sdp.split('\n');
  let line = -1;
  for (let i = 0; i <= lines.length; i++) {
    if (lines[i].indexOf('m=' + media) === 0) {
      line = i;
      break;
    }
  }
  if (line === -1) {
    console.debug('Could not find the m line for', media);
    return sdp;
  }
  console.debug('Found the m line for', media, 'at line', line);

  // Pass the m line
  line++;

  // Skip i and c lines
  while (lines[line].indexOf('i=') === 0 || lines[line].indexOf('c=') === 0) {
    line++;
  }

  // If we're on a b line, replace it
  if (lines[line].indexOf('b') === 0) {
    console.debug('Replaced b line at line', line);
    lines[line] = 'b=AS:' + bitrate;
    return lines.join('\n');
  }

  // Add a new b line
  console.debug('Adding new b line before line', line);
  let newLines = lines.slice(0, line);
  newLines.push('b=AS:' + bitrate);
  newLines = newLines.concat(lines.slice(line, lines.length));
  return newLines.join('\n');
}
