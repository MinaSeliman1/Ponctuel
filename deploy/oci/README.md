# Déploiement public gratuit sur OCI

Le Compose complet de Ponctuel a besoin d'une machine qui conserve les
volumes TimescaleDB et Redpanda. Le déploiement recommandé pour une démo
publique gratuite est une VM **Oracle Cloud Infrastructure Always Free** avec
Ubuntu ARM : 2 OCPU, 12 Go de mémoire et un volume de démarrage de 50 Go.

Cette procédure publie le dashboard en HTTP sur le port 80. Elle démarre par
défaut en mode `fixture`, donc sans clé STM et sans coût d'API. Les ports API
8080 et matcher 8082 ne doivent pas être ouverts dans le pare-feu OCI.

## 1. Créer la VM OCI

1. Crée un compte [OCI Free Tier](https://www.oracle.com/cloud/free/). Pendant
   l'inscription, choisis soigneusement la région principale. Oracle peut
   demander un numéro de téléphone et une carte pour vérifier l'identité; la
   carte n'est pas débitée sauf si tu passes volontairement à des ressources
   payantes.
2. Dans **Compute > Instances**, crée une instance avec :
   - image Ubuntu 24.04 ARM, marquée **Always Free Eligible**;
   - forme `VM.Standard.A1.Flex`;
   - exactement **2 OCPU** et **12 Go** de mémoire;
   - volume de démarrage de 50 Go;
   - une adresse IPv4 publique;
   - une clé SSH que tu conserves localement.
3. Dans la security list du subnet, autorise :
   - TCP `22` uniquement depuis ton adresse IP pour SSH;
   - TCP `80` depuis `0.0.0.0/0` pour le dashboard.

Ne crée pas de ressource marquée payante et n'ouvre pas 8080/8082.

## 2. Installer et démarrer Ponctuel

Depuis une session SSH sur la VM (`ubuntu` est le nom habituel avec Ubuntu) :

```bash
curl -fsSL https://raw.githubusercontent.com/MinaSeliman1/Ponctuel/jalon-1/deploy/oci/bootstrap.sh -o /tmp/ponctuel-bootstrap.sh
bash /tmp/ponctuel-bootstrap.sh
```

Le script installe Docker Compose, clone `jalon-1`, crée le `.env` local de la
VM et démarre tous les services. Le premier build peut prendre plusieurs
minutes sur une petite VM ARM.

Ouvre ensuite :

```text
http://IP-PUBLIQUE-DE-LA-VM
```

Le dashboard utilise alors les fixtures reproductibles. Les mises à jour
manuelles peuvent être faites avec :

```bash
bash "$HOME/Ponctuel/deploy/oci/deploy.sh"
```

## 3. Activer les données STM réelles (facultatif)

Sur la VM uniquement, édite `$HOME/Ponctuel/.env` et configure :

```dotenv
APP_ENV=stm
STM_API_KEY=ta_cle_stm
STM_API_KEY_HEADER=apikey
```

Puis relance le script de déploiement. Ne colle jamais la clé dans GitHub,
dans le frontend, dans une image Docker ou dans une issue publique.

## 4. Déploiement automatique depuis GitHub (facultatif)

Après le premier bootstrap, ajoute dans **Settings > Secrets and variables >
Actions** les secrets suivants :

| Secret | Valeur |
| --- | --- |
| `OCI_DEPLOY_HOST` | IP publique de la VM |
| `OCI_DEPLOY_USER` | `ubuntu` |
| `OCI_DEPLOY_SSH_KEY` | clé privée SSH de déploiement |
| `OCI_DEPLOY_KNOWN_HOSTS` | sortie de `ssh-keyscan -H IP-PUBLIQUE` |

Ensuite, lance **Actions > Deploy public OCI > Run workflow** depuis la
branche `jalon-1`. Le workflow ne reçoit pas la clé STM; elle reste seulement
dans le `.env` de la VM si le mode réel est activé.

## Limites honnêtes

OCI peut signaler un manque temporaire de capacité pour les petites VM ARM.
Les ressources Always Free peuvent aussi être récupérées si une VM reste
inutilisée pendant une période prolongée. Cette solution convient à une démo
publique et à un portfolio, pas à un SLA de production. Pour obtenir HTTPS et
un nom stable, ajoute ensuite un domaine personnel et un proxy TLS; l'adresse
IP HTTP suffit pour valider gratuitement le produit.
