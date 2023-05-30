require('dotenv').config();
const ethers = require('ethers');
const axios = require('axios');

const serverAddress = process.env.SERVER_ADDRESS;
const privateKey = process.env.PRIVATE_KEY;
const toAddress = process.env.TO_ADDRESS;

const waitBetweenOperations = 3000;
const waitBetweenTests = 10000;

const provider = new ethers.providers.JsonRpcProvider(serverAddress); 
const wallet = new ethers.Wallet(privateKey, provider);



// Sleep function.
async function wait(ms) {
    return new Promise(resolve => setTimeout(resolve, ms));
}

// transaction builder
function buildTransactionRaw(toAddress,value,isLowGas,nonce){
    return isLowGas ?
     {
        to: toAddress,
        value: value,
        maxPriorityFeePerGas: ethers.utils.parseUnits('0', 'gwei'),
        maxFeePerGas: ethers.utils.parseUnits('0', 'gwei'),
        nonce

    }:
    {
    to: toAddress,
    value: value,
    nonce
    }
    
}


// Send transaction function
async function sendTransaction(rawTx) {
        return await wallet.sendTransaction(rawTx);
}


// Cancel transaction throught custom RPC method.
async function cancelTransaction(hash) {
    try{
        const response = await axios.post(serverAddress, { 
            jsonrpc: '2.0',
            method: 'cancel_transaction',
            params: [hash],
            id: 1,
        });
    
        if (response.data.error) {
            console.error(`Failed to cancel transaction: ${response.data.error.message}`);
        } else {
            console.log('Transaction canceled successfully');
        }
    }catch(err){
        console.log(err)
    }

}


// Cancel transaction process with custom rpc method process.
async function cancelWithRPCMethodProcess() {
    try{
        const tx = await sendTransaction(buildTransactionRaw(toAddress,ethers.utils.parseEther('0.0004'),true ));
        console.log(`cancelWithRPCMethodProcess: transaction hash: ${tx.hash}`)
        await wait(waitBetweenOperations);
        await cancelTransaction(tx.hash);
    }catch(err){
        console.error(err)
    }

}

// Send already stored transaction process.
async function sendAlreadyStoredTransactionProcess() {
    try{
        tx  = await sendTransaction(buildTransactionRaw(toAddress,ethers.utils.parseEther('0.0001'),true ));
        console.log(`sendAlreadyStoredTransactionProcess: transaction hash: ${tx.hash}`)
        await sendTransaction(buildTransactionRaw(toAddress,ethers.utils.parseEther('0.0001'),true,tx.nonce ));
    }catch(err){
        console.log(`sendAlreadyStoredTransactionProcess: error: ${err.body}` )
    }
   

}


// Speeding up a transaction process.
async function speedUpProcess() {
    try{
        tx =  await sendTransaction(buildTransactionRaw(toAddress,ethers.utils.parseEther('0.0002'),true ));
        console.log(`speedUpProcess: transaction hash: ${tx.hash}`)
        await wait(waitBetweenOperations);
       speedUpTx =  await sendTransaction(buildTransactionRaw(toAddress,ethers.utils.parseEther('0.0002'),false, tx.nonce ));
        console.log(`speedUpProcess: This tx: ${tx.hash} has been replaced by this one: ${speedUpTx.hash}`)
        const txWait = setInterval(async ()=>{
            try{
                const txReceipt = await provider.getTransactionReceipt(speedUpTx.hash);
                console.log(`speedUpProcess: Speed up transaction blockNumber: ${txReceipt.blockNumber}`)
                clearInterval(txWait)
            }catch(err){
                console.error("speedUpProcess: Speed up transaction not executed yet")
            }
        } , 3000)
    }catch(err){
        console.error(err)
    }
}


// Cancel a transaction with another transaction process.
async function cancelWithTransactionProcess() {
    try{
    tx = await sendTransaction(buildTransactionRaw(toAddress,ethers.utils.parseEther('0.0003'),true ));
    console.log(`cancelWithTransactionProcess: transaction hash: ${tx.hash}`)
    await wait(waitBetweenOperations);
    cancelTx = await sendTransaction(buildTransactionRaw(wallet.address,ethers.utils.parseEther('0'),false, tx.nonce ));
    console.log(`cancelWithTransactionProcess: This tx: ${tx.hash} has been canceled by this one: ${cancelTx.hash}, none of them submitted`)
    }catch(err){
        console.error(err)
    }

}

// Send a transaction and wait for its confirmation process.
async function submitTransactionProcess() {
    try{
        const tx = await sendTransaction(buildTransactionRaw(toAddress,ethers.utils.parseEther('0.0001'),false ));
        console.log(`submitTransactionProcess: transaction hash: ${tx.hash}`)
        const txWait = setInterval(async ()=>{
            try{
                const txReceipt = await provider.getTransactionReceipt(tx.hash);
                console.log(`submitTransactionProcess: transaction blockNumber: ${txReceipt.blockNumber}`)
                clearInterval(txWait)
            }catch(err){
                console.error("submitTransactionProcess: Transaction not executed yet")
            }
          
        } , 3000)
    }catch(err){
        console.error(err)
    }

  
}



// Main function.
async function main(){
    try{
        await cancelWithRPCMethodProcess()
        await wait(waitBetweenTests); 
        await cancelWithTransactionProcess()
        await  wait(waitBetweenTests);
        await  speedUpProcess()
        await  wait(waitBetweenTests);
        await submitTransactionProcess()
        await  wait(waitBetweenTests);

        // this will return an error since the server return an error in case of an existing transaction. and ethereum node will return already known.
        await sendAlreadyStoredTransactionProcess()
    }catch(err){
        console.error(`Main function error: ${err}`);
    }
}

main()
