module Pages.Home_ exposing (Model, Msg, page)

import Api.Data as Data exposing (Data)
import Api.Node as Node exposing (Node, NodeView)
import Api.Point as Point exposing (Point)
import Api.Response exposing (Response)
import Auth
import Components.NodeAction as NodeAction
import Components.NodeCanBus as NodeCanBus
import Components.NodeCondition as NodeCondition
import Components.NodeDb as NodeDb
import Components.NodeDevice as NodeDevice
import Components.NodeFile as File
import Components.NodeGroup as NodeGroup
import Components.NodeMessageService as NodeMessageService
import Components.NodeModbus as NodeModbus
import Components.NodeModbusIO as NodeModbusIO
import Components.NodeOneWire as NodeOneWire
import Components.NodeOneWireIO as NodeOneWireIO
import Components.NodeOptions exposing (CopyMove(..), NodeOptions)
import Components.NodeRule as NodeRule
import Components.NodeSerialDev as NodeSerialDev
import Components.NodeSignalGenerator as SignalGenerator
import Components.NodeSync as NodeSync
import Components.NodeUser as NodeUser
import Components.NodeVariable as NodeVariable
import Dict
import Effect exposing (Effect)
import Element exposing (..)
import Element.Background as Background
import Element.Font as Font
import Element.Input as Input
import File
import File.Select
import Gen.Params.Home_ exposing (Params)
import List.Extra
import Page
import Request
import Shared
import Storage
import Task
import Time
import Tree exposing (Tree)
import Tree.Zipper as Zipper
import UI.Button as Button
import UI.Form as Form
import UI.Icon as Icon
import UI.Layout
import UI.Style as Style exposing (colors)
import UI.ViewIf exposing (viewIf)
import View exposing (View)


page : Shared.Model -> Request.With Params -> Page.With Model Msg
page shared req =
    Page.protected.advanced <|
        \user ->
            { init = init shared
            , update = update shared
            , view = view user shared
            , subscriptions = subscriptions
            }



-- INIT


type alias Model =
    { nodeEdit : Maybe NodeEdit
    , zone : Time.Zone
    , now : Time.Posix
    , nodes : List (Tree NodeView)
    , error : Maybe String
    , nodeOp : NodeOperation
    , copyMove : CopyMove
    , nodeMsg : Maybe NodeMsg
    }


type alias NodeMsg =
    { feID : Int
    , text : String
    , time : Time.Posix
    }


type NodeOperation
    = OpNone
    | OpNodeToAdd NodeToAdd
    | OpNodeMessage NodeMessage
    | OpNodeDelete Int String String
    | OpNodePaste Int String


type alias NodeEdit =
    { feID : Int
    , points : List Point
    }


type alias NodeToAdd =
    { typ : Maybe String
    , feID : Int
    , parent : String
    }


type alias NodeMessage =
    { feID : Int
    , id : String
    , parent : String
    , message : String
    }


defaultModel : Model
defaultModel =
    Model
        Nothing
        Time.utc
        (Time.millisToPosix 0)
        []
        Nothing
        OpNone
        CopyMoveNone
        Nothing


init : Shared.Model -> ( Model, Effect Msg )
init shared =
    let
        model =
            defaultModel

        token =
            case shared.storage.user of
                Just user ->
                    user.token

                Nothing ->
                    ""
    in
    ( model
    , Effect.fromCmd <|
        Cmd.batch
            [ Task.perform Zone Time.here
            , Task.perform Tick Time.now
            , Node.list { onResponse = ApiRespList, token = token }
            ]
    )



-- UPDATE


type Msg
    = SignOut
    | Tick Time.Posix
    | Zone Time.Zone
    | EditNodePoint Int (List Point)
    | UploadFile String
    | UploadSelected String File.File
    | UploadContents String File.File String
    | ToggleExpChildren Int
    | ToggleExpDetail Int
    | DiscardNodeOp
    | DiscardEdits
    | AddNode Int String
    | MsgNode Int String String
    | PasteNode Int String
    | DeleteNode Int String String
    | UpdateMsg String
    | SelectAddNodeType String
    | ApiDelete String String
    | ApiPostPoints String
    | ApiPostAddNode Int
    | ApiPostMoveNode Int String String String
    | ApiPutMirrorNode Int String String
    | ApiPutDuplicateNode Int String String
    | ApiPostNotificationNode
    | ApiRespList (Data (List Node))
    | ApiRespDelete (Data Response)
    | ApiRespPostPoint (Data Response)
    | ApiRespPostAddNode Int (Data Response)
    | ApiRespPostMoveNode Int (Data Response)
    | ApiRespPutMirrorNode Int (Data Response)
    | ApiRespPutDuplicateNode Int (Data Response)
    | ApiRespPostNotificationNode (Data Response)
    | CopyNode Int String String String
    | ClearClipboard


update : Shared.Model -> Msg -> Model -> ( Model, Effect Msg )
update shared msg model =
    case msg of
        SignOut ->
            ( model, Effect.fromCmd <| Storage.signOut shared.storage )



-- SUBSCRIPTIONS


subscriptions : Model -> Sub Msg
subscriptions model =
    Sub.none



-- VIEW


view : Auth.User -> Shared.Model -> Model -> View Msg
view user shared model =
    { title = "SIOT"
    , attributes = []
    , element =
        UI.Layout.layout
            { onSignOut = SignOut
            , email =
                case shared.storage.user of
                    Just user_ ->
                        Just user_.email

                    Nothing ->
                        Nothing
            , error = shared.error
            }
            (text "Home_")
    }
