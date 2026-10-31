import { Link } from "react-router-dom";
import { Brand } from "../components/ui/Brand";
import { formatPrice } from "../products/money";
import "./Home.css";

const capabilities = [
  { number: "01", title: "Catálogo organizado", text: "Cadastre produtos, atualize nomes e preços e encontre os itens por busca. A gestão fica com o administrador." },
  { number: "02", title: "Pedidos consistentes", text: "Monte o carrinho, ajuste quantidades e confira o total. Uma tentativa pendente pode ser retomada sem criar outro pedido." },
  { number: "03", title: "Acesso por perfil", text: "Operadores registram pedidos; administradores gerenciam o catálogo. A API restringe a consulta de pedidos ao responsável ou a um administrador." },
];

export function Home() {
  return <div className="home-page" id="inicio">
    <a className="skip-link" href="#home-content">Pular para o conteúdo</a>
    <header className="home-header"><div className="home-container home-header__inner">
      <a href="#inicio" className="brand-link" aria-label="TechSnacks, início da apresentação"><Brand /></a>
      <nav aria-label="Navegação da apresentação" className="home-nav"><a href="#sobre">Sobre</a><a href="#funcionalidades">Funcionalidades</a><a href="#tecnologia">Tecnologia</a></nav>
      <Link className="button button--primary" to="/login">Entrar no sistema</Link>
    </div></header>

    <main id="home-content" tabIndex={-1}>
      <section className="home-container home-hero" aria-labelledby="home-title">
        <div className="home-hero__copy"><span className="eyebrow">TECHSNACKS · PROJETO FULL STACK</span>
          <h1 id="home-title">Do catálogo ao pedido,<br /><em>um caixa com clareza.</em></h1>
          <p>Um projeto de ponto de venda que une uma operação simples a regras de negócio bem definidas. Catálogo, pedidos e controle de acesso em uma aplicação feita com Go e React.</p>
          <div className="home-actions"><Link to="/login" className="button button--primary">Entrar no sistema <span aria-hidden="true">↗</span></Link><a className="button button--secondary" href="#sobre">Conhecer o projeto</a></div>
          <p className="home-access-note">A apresentação é pública. Para usar o sistema, é necessário ter uma conta criada pelo administrador.</p>
          <ul className="home-tech-tags" aria-label="Tecnologias principais"><li>Go</li><li>React</li><li>TypeScript</li><li>PostgreSQL</li></ul>
        </div>
        <figure className="home-preview">
          <div className="home-preview__screen" aria-hidden="true">
            <div className="home-preview__top"><span className="home-preview__dots"><i /><i /><i /></span><span>TechSnacks / caixa</span></div>
            <div className="home-preview__heading"><span>Frente de caixa</span><span className="badge">Exemplo</span></div>
            <div className="home-preview__layout"><div className="home-preview__catalog"><span className="home-preview__label">CATÁLOGO</span>
              <div className="home-preview__product"><span>CF</span><div><strong>Coxinha de frango</strong><small>{formatPrice(650)}</small></div></div>
              <div className="home-preview__product"><span>PQ</span><div><strong>Pão de queijo</strong><small>{formatPrice(350)}</small></div></div>
              <div className="home-preview__hint">Escolher. Conferir. Registrar.</div>
            </div><div className="home-preview__order"><span className="home-preview__label">PEDIDO ATUAL</span><p>Coxinha de frango <strong>× 1</strong></p><p>Pão de queijo <strong>× 1</strong></p><div className="home-preview__total"><small>Total</small><strong>{formatPrice(1000)}</strong></div><span className="home-preview__status">Aguardando pagamento</span></div></div>
          </div>
          <figcaption>Prévia ilustrativa do caixa, com dados fictícios. Não representa uma venda real.</figcaption>
        </figure>
      </section>

      <section id="sobre" className="home-section home-about" aria-labelledby="about-title"><div className="home-container home-about__layout">
        <div><span className="eyebrow">SOBRE O PROJETO</span><h2 id="about-title">Uma interface simples.<br />Uma base técnica cuidadosa.</h2></div>
        <div className="home-section__text"><p>O TechSnacks foi pensado para organizar o atendimento de estabelecimentos alimentícios: encontrar produtos, montar um pedido e manter seus registros consistentes.</p><p>Também é um projeto de engenharia de software. A interface em React conversa com uma API em Go, enquanto o PostgreSQL guarda usuários, sessões, catálogo e pedidos. Cada camada tem uma responsabilidade clara.</p><a href="#tecnologia" className="home-text-link">Entenda a arquitetura <span aria-hidden="true">→</span></a></div>
      </div></section>

      <section id="funcionalidades" className="home-container home-section" aria-labelledby="features-title">
        <div className="home-section__heading"><span className="eyebrow">O QUE JÁ ESTÁ IMPLEMENTADO</span><h2 id="features-title">O essencial da operação,<br />com atenção aos detalhes.</h2><p>Fluxos de catálogo e pedidos com permissões verificadas pelo backend.</p></div>
        <div className="home-capabilities">{capabilities.map(item => <article key={item.number} className="home-capability"><span className="home-capability__number" aria-hidden="true">{item.number}</span><h3>{item.title}</h3><p>{item.text}</p></article>)}</div>
        <div className="home-flow"><div><span className="eyebrow">COMO FUNCIONA</span><h3>Um pedido, em três etapas.</h3></div><ol><li><span aria-hidden="true">1</span> Escolha os produtos</li><li><span aria-hidden="true">2</span> Confira itens e total</li><li><span aria-hidden="true">3</span> Registre o pedido</li></ol></div>
      </section>

      <section id="tecnologia" className="home-section home-technology" aria-labelledby="technology-title"><div className="home-container home-technology__layout">
        <div><span className="eyebrow">POR TRÁS DA INTERFACE</span><h2 id="technology-title">Tecnologia com propósito.<br />Responsabilidades separadas.</h2><p>Uma aplicação web com arquitetura em camadas, valores monetários em centavos e respostas tipadas na integração.</p>
          <dl className="home-stack"><div><dt>Frontend</dt><dd>React · TypeScript · Vite</dd></div><div><dt>Backend</dt><dd>Go · Gin · GORM · PostgreSQL</dd></div><div><dt>Qualidade e execução</dt><dd>Testes Go · Playwright · Docker · GitHub Actions</dd></div></dl>
        </div>
        <div className="home-architecture" aria-label="Camadas da arquitetura"><span className="home-architecture__caption">SEPARAÇÃO DE RESPONSABILIDADES</span>
          <div><span>01 / Domínio</span><h3>As regras do negócio.</h3><p>Produtos, estados do pedido e cálculos monetários.</p></div>
          <div><span>02 / Aplicação</span><h3>Os casos de uso.</h3><p>Catálogo, checkout, autenticação e autorização.</p></div>
          <div><span>03 / Infraestrutura</span><h3>As integrações.</h3><p>API HTTP, persistência e implementações de repositórios.</p></div>
        </div>
      </div></section>

      <section className="home-container home-section home-evolution" aria-labelledby="evolution-title"><div><span className="eyebrow">ESCOPO ATUAL</span><h2 id="evolution-title">Um projeto em evolução,<br />com limites explícitos.</h2><p>O checkout registra um pedido <strong>aguardando pagamento</strong>. Registrar um pedido não confirma seu recebimento.</p></div>
        <div className="home-evolution__details"><h3>Próximas evoluções de produto</h3><ul><li>Confirmação de recebimento e integrações de pagamento.</li><li>Controle de estoque físico e movimentações.</li><li>Fechamento de caixa e relatórios operacionais.</li></ul><p>Há testes automatizados e configuração de CI versionados. A homologação das imagens e do ambiente de produção depende da infraestrutura de execução.</p></div>
      </section>

      <section className="home-container home-access" aria-labelledby="access-title"><div><span className="eyebrow">CONHEÇA A APLICAÇÃO</span><h2 id="access-title">Da apresentação para a operação.</h2><p>Já tem uma conta? Acesse o caixa com o seu perfil.</p></div><Link to="/login" className="button button--primary">Entrar no sistema <span aria-hidden="true">↗</span></Link></section>
    </main>
    <footer className="home-container home-footer"><Brand /><p>Projeto de ponto de venda com Go e React.</p><a href="#inicio">Voltar ao topo ↑</a></footer>
  </div>;
}
