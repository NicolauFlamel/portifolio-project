# Gestão Orçamentária Pública com Blockchain
Transparência • Auditoria • Rastreabilidade • Imutabilidade

Este repositório contém o sistema desenvolvido como projeto acadêmico para garantir transparência, integridade e auditoria nos dados públicos entre União, Estados e Municípios, utilizando Hyperledger Fabric, Go (Gin) e Next.js.

Embora o foco inicial seja auditoria de repasses financeiros, a arquitetura foi projetada para registrar e validar qualquer documento governamental estruturado em JSON — incluindo licitações, contratos, convênios, relatórios e prestações de contas.

Assim, o sistema funciona como uma camada de verificação e integridade sobre sistemas públicos já existentes, sem substituí-los.

A documentação completa está disponível em:

- 📘 [1 — Visão Geral]
- 🧱 [2 — Arquitetura do Sistema]
- 🏗️ [3 — Arquitetura da Blockchain]
- 🧩 [4 — Diagramas C4]
- 📋 [5 — Requisitos Funcionais]
- 📁 [6 — Estrutura do Repositório]
- 🚀 [7 — Guia de Deploy]
- 🛠 [8 — Tecnologias Utilizadas]

# 1. Visão Geral

Os sistemas públicos brasileiros sofrem com fragmentação estrutural: União, Estados e Municípios operam com bancos de dados isolados, modelos próprios de gestão e aplicações que não se comunicam entre si. Essa falta de integração causa:
- divergências entre dados de diferentes esferas,´
- atrasos e inconsistências nos repasses
- dificuldade de auditoria
- ausência de rastreabilidade confiável
- risco de adulteração ou perda de integridade
- falta de transparência para o cidadão e para órgãos de controle

Esses problemas não decorrem apenas de falhas humanas, mas de uma arquitetura governamental onde cada esfera mantém sistemas centralizados e desconectados, dificultando verificações cruzadas e auditorias independentes.

## Objetivos

Diante desse cenário, este projeto propõe uma camada de integridade baseada em blockchain permissionada para unificar a verificação de dados públicos entre União, Estados e Municípios, sem substituir os sistemas atuais.

A solução utiliza Hyperledger Fabric com coleções privadas, permitindo que cada esfera registre documentos estruturados (JSON) — incluindo repasses financeiros, contratos, licitações, relatórios e outros artefatos governamentais — de forma:
- Imutável
- Auditável
- Assinada digitalmente
- Privada quando necessário
- Integrada com todas as esferas governamentais

Dessa forma, o sistema resolve o problema central:
criar um ambiente confiável de interoperabilidade e verificação entre as três esferas governamentais, eliminando divergências e restaurando a integridade compartilhada dos dados públicos.

# 2. Arquitetura do Sistema

A arquitetura do sistema adota o modelo client–server integrado a uma blockchain permissionada. Ela é composta por três camadas principais: Frontend, Backend e Hyperledger Fabric. Cada uma desempenha um papel específico e desacoplado, garantindo organização, segurança e evolutividade do projeto.

## 2.1 Visão Geral em Camadas

A comunicação segue fluxo descendente:

**TODO= DIAGRAMA GERAL**

1. O usuário interage com o frontend
2. O frontend envia requisições ao backend via API REST
3. O backend realiza validações, autenticação, hashing e encaminha a operação para a blockchain através do Fabric SDK
4. O Hyperledger Fabric executa as regras do chaincode e persiste dados públicos ou privados conforme as coleções definidas

## 2.2 Frontend

O frontend é responsável por:
- Oferecer uma interface clara para auditores e servidores públicos,
- Exibir repasses, documentos, inconsistências e histórico,
- Realizar chamadas seguras ao backend,
- Organizar filtros, buscas e dashboards.

Características:
- Totalmente desacoplado do Fabric
- Não possui lógica de negócio sensível
- Não acessa o blockchain diretamente
- Interage exclusivamente via API REST

## 2.3 Backend (Go + Gin)

O backend atua como gateway, validador de neg[ocios e cliente oficial da blockchain. Responsabilidades incluem:
- API REST:
  - Rotas para criação e consulta de documentos e repasses
  - Autenticação/autorização (se aplicável)
  - Respostas padronizadas no formato JSON
  - Documentação dos endpoints no Swagger
- Lógica de Negócio:
  - Validações pré-transação
  - Versões de documentos
  - Tipo de documento (financeiro, licitação, relatório, etc)
  - Seleção automática da coleção privada correta
  - Hashing e auditoria
- Comunicação com Fabric via SDK:
  - Submit/evaluate de transactions
  - Envio de JSONs
  - Acesso a coleções privadas
  - Tratamento de endorsements e erros

## 2.4 Blockchain

A camada blockchain é responsável por:
- Imutabilidade
- Auditoria
- Validação de transações
- Verificação de assinaturas
- Execução de chaincode determinístico
 

# 3. Arquitetura Blockchain


