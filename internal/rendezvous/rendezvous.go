package rendezvous

// ouve tcp:
//   recebe um pedido de discovery -> responde yes
//   recebe um stream request ->
//      verifica se esta a difundir esse conteudo 
//          yes: responde com a porta da stream 
//          no: comunicar com o server que tiver o conteudo e comecar a stream
//              e responde com a porta
//          para ambos espera que o nodo respoda com ok antes de abrir o udp 

type Rendezvous struct {
    // map key server addr e value struct com struct de metricas conteudo associado
    // ao server e.g. {Fonte:S1; Métrica: 1; Conteúdos: [movie1.mp4, video4.ogg], Estado: ativa }
    
}
