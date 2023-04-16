<h1 align="center"> 

CamOpen - SJCam SJ4000Air Wifi

</h1>

***SJ4000 Air 4k Wifi***


## Um programa para conectar minha câmera de ação SJcam 4000Air e capturar feeds ao vivo


### Informação
Este programa se conecta à minha câmera de ação sjcam 4000, busca transmissão ao vivo (dados h264 sem cabeça brutos) e os despeja no diretório de trabalho atual.

<!--

Eu recebi uma grande ajuda desta postagem no blog [Hacking the TecTecTec! XPro2 Action Camera](https://blog.jonaskoeritz.de/2017/02/21/hacking-the-xpro2-action-camera/) por Jonas Köritz, e seu projeto github [actioncam](https://github. com/jonas-koeritz/actioncam).

-->

Escolhi usar o Rust porque me sentia mais confortável com ele do que com o Go.

Ele poderia ter implementado um feed RTSP, mas meu objetivo para este projeto era gravar a transmissão ao vivo do meu computador sem inserir um cartão SD na câmera. Bem, meio que funciona, mas estou fazendo poucas pesquisas sobre como converter os dados h264 brutos em qualquer formato de mídia universal, como MKV, MP4 ou WAV. Vai levar algum tempo para eu descobrir isso.

### Uso

* `cargo run .` (inicia o servidor receptor) (começa a salvar o stream como `xcal.h264`)
* `./playvlc` (reproduzir a transmissão ao vivo com vlc [a transmissão tem cerca de 4-5 segundos de atraso]) (Você precisa ter o VLC media player instalado)
* `./convert2mp4` (converte o stream salvo em arquivo `abcd.mp4`) (**Por favor, interrompa o aplicativo de ferrugem antes de converter**, pois o script nunca terminará o trabalho) (Você precisa ter o ffmpeg instalado com h264 codecs)
