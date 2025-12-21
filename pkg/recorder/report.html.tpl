<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <title>Tend TUI Session Recording</title>
    <style>
        body { font-family: monospace; background-color: #1e1e1e; color: #d4d4d4; margin: 0; display: flex; height: 100vh; }
        #sidebar { width: 300px; background-color: #252526; padding: 10px; overflow-y: auto; border-right: 1px solid #333; }
        #main { flex-grow: 1; padding: 20px; display: flex; flex-direction: column; }
        #terminal { background-color: #000; padding: 10px; border-radius: 5px; white-space: pre; overflow-x: auto; flex-grow: 1; }
        .frame-item { padding: 8px; cursor: pointer; border-radius: 4px; margin-bottom: 5px; border-left: 3px solid transparent; }
        .frame-item:hover { background-color: #333; }
        .frame-item.active { background-color: #007acc; border-left-color: #fff; }
        .frame-input { font-weight: bold; color: #4ec9b0; }
        #controls { margin-bottom: 20px; }
        button { background: #007acc; color: white; border: none; padding: 8px 15px; border-radius: 5px; cursor: pointer; margin-right: 10px; }
        button:hover { background: #005a9e; }
    </style>
</head>
<body>
    <div id="sidebar">
        <h3>Session Timeline</h3>
        {{range $i, $frame := .Frames}}
        <div class="frame-item" data-index="{{$i}}">
            <div class="frame-ts">{{$frame.Timestamp}}</div>
            <div class="frame-input">Input: {{$frame.Input}}</div>
        </div>
        {{end}}
    </div>
    <div id="main">
        <div id="controls">
            <button id="play">▶ Play</button>
            <button id="prev">« Prev</button>
            <button id="next">Next »</button>
            <span id="current-frame">Frame: 0 / {{len .Frames}}</span>
        </div>
        <div id="terminal"></div>
    </div>

    <div id="snapshots" style="display: none;">
        {{range .Frames}}<div class="snapshot">{{.Snapshot}}</div>{{end}}
    </div>

    <script>
        const frames = document.querySelectorAll('.frame-item');
        const snapshots = document.querySelectorAll('.snapshot');
        const terminal = document.getElementById('terminal');
        const currentFrameLabel = document.getElementById('current-frame');
        let currentIndex = -1;
        let playbackInterval;

        function showFrame(index) {
            if (index < 0 || index >= frames.length) return;

            if (currentIndex !== -1) {
                frames[currentIndex].classList.remove('active');
            }
            currentIndex = index;
            frames[currentIndex].classList.add('active');
            terminal.innerHTML = snapshots[currentIndex].innerHTML;
            currentFrameLabel.textContent = `Frame: ${currentIndex + 1} / ${frames.length}`;
            frames[currentIndex].scrollIntoView({ block: 'center' });
        }

        frames.forEach((frame, i) => {
            frame.addEventListener('click', () => {
                stopPlayback();
                showFrame(i);
            });
        });

        document.getElementById('next').addEventListener('click', () => {
            stopPlayback();
            showFrame(currentIndex + 1);
        });

        document.getElementById('prev').addEventListener('click', () => {
            stopPlayback();
            showFrame(currentIndex - 1);
        });

        const playBtn = document.getElementById('play');
        playBtn.addEventListener('click', () => {
            if (playbackInterval) {
                stopPlayback();
            } else {
                startPlayback();
            }
        });

        function startPlayback() {
            playBtn.textContent = '❚❚ Pause';
            playbackInterval = setInterval(() => {
                if (currentIndex >= frames.length - 1) {
                    stopPlayback();
                } else {
                    showFrame(currentIndex + 1);
                }
            }, 500);
        }

        function stopPlayback() {
            clearInterval(playbackInterval);
            playbackInterval = null;
            playBtn.textContent = '▶ Play';
        }

        showFrame(0);
    </script>
</body>
</html>
